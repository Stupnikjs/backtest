package api

var query string = `{
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
