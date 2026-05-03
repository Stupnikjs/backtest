package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math/big"
    "net/http"
    "os"
    "os/exec"
    "time"

    "github.com/lmittmann/w3"
    "github.com/lmittmann/w3/module/eth"
    "github.com/lmittmann/w3/w3types"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/common/hexutil"
)

// ─── Morpho API ───────────────────────────────────────────────

type MorphoLiquidation struct {
    BlockNumber     uint64
    TxHash          string
    Borrower        common.Address
    MarketUniqueKey string
    SeizedAssets    *big.Int
    RepaidAssets    *big.Int
    Market          MarketInfo
}

type MarketInfo struct {
    LoanToken       common.Address
    CollateralToken common.Address
    Oracle          common.Address
    IRM             common.Address
    LLTV            *big.Int
}

func fetchMorphoLiquidations(fromBlock, toBlock uint64) ([]MorphoLiquidation, error) {
    query := `{
        "query": "{ 
            transactions(
                where: { 
                    type: { eq: \"Liquidation\" }
                    blockNumber: { gte: %d, lte: %d }
                }
                orderBy: BlockNumber
                orderByDirection: ASC
                first: 100
            ) { 
                items { 
                    blockNumber 
                    hash
                    data { 
                        ... on MorphoBlueTransactionData { 
                            market { 
                                uniqueKey
                                loanAsset { address }
                                collateralAsset { address }
                                oracle { address }
                                irm { address }
                                lltv
                            }
                            borrower { address }
                            seizedAssets
                            repaidAssets
                        } 
                    } 
                } 
            } 
        }"
    }`

    body := fmt.Sprintf(query, fromBlock, toBlock)

    resp, err := http.Post(
        "https://blue-api.morpho.org/graphql",
        "application/json",
        bytes.NewBufferString(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data struct {
            Transactions struct {
                Items []struct {
                    BlockNumber uint64 `json:"blockNumber"`
                    Hash        string `json:"hash"`
                    Data        struct {
                        Market struct {
                            UniqueKey      string `json:"uniqueKey"`
                            LoanAsset      struct{ Address string `json:"address"` }
                            CollateralAsset struct{ Address string `json:"address"` }
                            Oracle         struct{ Address string `json:"address"` }
                            IRM            struct{ Address string `json:"address"` }
                            LLTV           string `json:"lltv"`
                        }
                        Borrower     struct{ Address string `json:"address"` }
                        SeizedAssets string `json:"seizedAssets"`
                        RepaidAssets string `json:"repaidAssets"`
                    }
                }
            }
        }
    }{}

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    var liquidations []MorphoLiquidation
    for _, item := range result.Data.Transactions.Items {
        lltv, _ := new(big.Int).SetString(item.Data.Market.LLTV, 10)
        seized, _ := new(big.Int).SetString(item.Data.SeizedAssets, 10)
        repaid, _ := new(big.Int).SetString(item.Data.RepaidAssets, 10)

        liquidations = append(liquidations, MorphoLiquidation{
            BlockNumber:     item.BlockNumber,
            TxHash:          item.Hash,
            Borrower:        common.HexToAddress(item.Data.Borrower.Address),
            MarketUniqueKey: item.Data.Market.UniqueKey,
            SeizedAssets:    seized,
            RepaidAssets:    repaid,
            Market: MarketInfo{
                LoanToken:       common.HexToAddress(item.Data.Market.LoanAsset.Address),
                CollateralToken: common.HexToAddress(item.Data.Market.CollateralAsset.Address),
                Oracle:          common.HexToAddress(item.Data.Market.Oracle.Address),
                IRM:             common.HexToAddress(item.Data.Market.IRM.Address),
                LLTV:            lltv,
            },
        })
    }

    return liquidations, nil
}

// ─── Anvil ────────────────────────────────────────────────────

type AnvilInstance struct {
    Port    int
    Cmd     *exec.Cmd
    RPC     string
    Client  *w3.Client
}

func startAnvil(forkURL string, blockNumber uint64, port int) (*AnvilInstance, error) {
    cmd := exec.Command("anvil",
        "--fork-url", forkURL,
        "--fork-block-number", fmt.Sprintf("%d", blockNumber-1), // juste avant la liquidation
        "--port", fmt.Sprintf("%d", port),
        "--no-mining",      // mining manuel = contrôle total
        "--silent",
    )
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("anvil start failed: %w", err)
    }

    // Attendre qu'Anvil soit prêt
    rpc := fmt.Sprintf("http://localhost:%d", port)
    client, err := waitForAnvil(rpc, 10*time.Second)
    if err != nil {
        cmd.Process.Kill()
        return nil, err
    }

    return &AnvilInstance{
        Port:   port,
        Cmd:    cmd,
        RPC:    rpc,
        Client: client,
    }, nil
}

func waitForAnvil(rpc string, timeout time.Duration) (*w3.Client, error) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        client, err := w3.Dial(rpc)
        if err == nil {
            var blockNum big.Int
            if err := client.Call(eth.BlockNumber().Returns(&blockNum)); err == nil {
                return client, nil
            }
        }
        time.Sleep(200 * time.Millisecond)
    }
    return nil, fmt.Errorf("anvil timeout after %s", timeout)
}

func (a *AnvilInstance) Stop() {
    a.Client.Close()
    a.Cmd.Process.Kill()
}

// ─── Test de liquidation ──────────────────────────────────────

type LiquidationResult struct {
    Liquidation MorphoLiquidation
    Success     bool
    Profit      *big.Int
    GasUsed     uint64
    Error       string
}

func testLiquidation(
    anvil *AnvilInstance,
    liq MorphoLiquidation,
    liquidatorContract common.Address,
    ownerAddr common.Address,
    swapRouter common.Address,
    poolFee uint32,
) LiquidationResult {

    result := LiquidationResult{Liquidation: liq}

    // Balance loan token avant
    balanceBefore := getTokenBalance(anvil.Client, liq.Market.LoanToken, ownerAddr)

    // Impersonate le owner (Anvil permet ça sans clé privée)
    anvil.impersonate(ownerAddr)

    // Encoder l'appel
    liquidateData := encodeLiquidateCall(
        liq.Market,
        liq.Borrower,
        liq.SeizedAssets,
        swapRouter,
        poolFee,
    )

    // Envoyer la tx
    txHash, gasUsed, err := anvil.sendTx(
        ownerAddr,
        liquidatorContract,
        liquidateData,
    )
    if err != nil {
        result.Error = err.Error()
        return result
    }

    _ = txHash
    result.GasUsed = gasUsed

    // Balance après + sweep
    anvil.sweep(ownerAddr, liquidatorContract, liq.Market.LoanToken)
    balanceAfter := getTokenBalance(anvil.Client, liq.Market.LoanToken, ownerAddr)

    profit := new(big.Int).Sub(balanceAfter, balanceBefore)
    result.Success = profit.Sign() > 0
    result.Profit = profit

    return result
}

func (a *AnvilInstance) impersonate(addr common.Address) {
    // anvil_impersonateAccount
    payload := map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "anvil_impersonateAccount",
        "params":  []string{addr.Hex()},
        "id":      1,
    }
    body, _ := json.Marshal(payload)
    http.Post(a.RPC, "application/json", bytes.NewReader(body))
}

// ─── Pipeline principal ───────────────────────────────────────

func main() {
    forkURL := os.Getenv("ALCHEMY_RPC")
    liquidatorContract := common.HexToAddress(os.Getenv("LIQUIDATOR_CONTRACT"))
    ownerAddr := common.HexToAddress(os.Getenv("OWNER_ADDR"))
    swapRouter := common.HexToAddress("0xE592427A0AEce92De3Edee1F18E0157C05861564")

    // Blocs du crash octobre 2025
    // ~21_350_000 à vérifier avec cast
    fromBlock := uint64(21_350_000)
    toBlock := uint64(21_360_000)

    fmt.Printf("Fetching liquidations blocs %d → %d...\n", fromBlock, toBlock)

    liquidations, err := fetchMorphoLiquidations(fromBlock, toBlock)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d liquidations\n", len(liquidations))

    var results []LiquidationResult
    port := 8545

    for i, liq := range liquidations {
        fmt.Printf("\n[%d/%d] Bloc %d — Borrower %s\n",
            i+1, len(liquidations),
            liq.BlockNumber,
            liq.Borrower.Hex(),
        )

        // Lancer Anvil forké au bloc N-1
        anvil, err := startAnvil(forkURL, liq.BlockNumber, port)
        if err != nil {
            fmt.Printf("  ✗ Anvil failed: %v\n", err)
            continue
        }

        res := testLiquidation(
            anvil,
            liq,
            liquidatorContract,
            ownerAddr,
            swapRouter,
            3000, // pool fee
        )

        if res.Success {
            fmt.Printf("  ✓ Profit: %s tokens | Gas: %d\n",
                res.Profit.String(), res.GasUsed)
        } else {
            fmt.Printf("  ✗ Failed: %s\n", res.Error)
        }

        results = append(results, res)
        anvil.Stop()

        port++ // port différent pour chaque instance
    }

    // Rapport final
    printReport(results)
}

func printReport(results []LiquidationResult) {
    var successes, failures int
    totalProfit := new(big.Int)

    for _, r := range results {
        if r.Success {
            successes++
            totalProfit.Add(totalProfit, r.Profit)
        } else {
            failures++
        }
    }

    fmt.Printf("\n═══ RAPPORT ═══\n")
    fmt.Printf("Total testées : %d\n", len(results))
    fmt.Printf("Succès        : %d\n", successes)
    fmt.Printf("Échecs        : %d\n", failures)
    fmt.Printf("Profit total  : %s\n", totalProfit.String())
}



