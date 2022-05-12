package consts

const BridgeABI = "[{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"domainID\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiry\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"destinationDomainID\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"resourceID\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"depositNonce\",\"type\":\"uint64\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"handlerResponse\",\"type\":\"bytes\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"EndKeygen\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"lowLevelData\",\"type\":\"bytes\"}],\"name\":\"FailedHandlerExecution\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"KeyRefresh\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"originDomainID\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"depositNonce\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"ProposalExecution\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"StartKeygen\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_MPCAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"name\":\"_depositCounts\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_domainID\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_expiry\",\"outputs\":[{\"internalType\":\"uint40\",\"name\":\"\",\"type\":\"uint40\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"_fee\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"_resourceIDToHandlerAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"getRoleMember\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleMemberCount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"getRoleMemberIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isValidForwarder\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"usedNonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newAdmin\",\"type\":\"address\"}],\"name\":\"renounceAdmin\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"adminPauseTransfers\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"adminUnpauseTransfers\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"handlerAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"resourceID\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"}],\"name\":\"adminSetResource\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"handlerAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"resourceID\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"depositFunctionSig\",\"type\":\"bytes4\"},{\"internalType\":\"uint256\",\"name\":\"depositFunctionDepositerOffset\",\"type\":\"uint256\"},{\"internalType\":\"bytes4\",\"name\":\"executeFunctionSig\",\"type\":\"bytes4\"}],\"name\":\"adminSetGenericResource\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"handlerAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"}],\"name\":\"adminSetBurnable\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"domainID\",\"type\":\"uint8\"},{\"internalType\":\"uint64\",\"name\":\"nonce\",\"type\":\"uint64\"}],\"name\":\"adminSetDepositNonce\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"forwarder\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"valid\",\"type\":\"bool\"}],\"name\":\"adminSetForwarder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newFee\",\"type\":\"uint256\"}],\"name\":\"adminChangeFee\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"handlerAddress\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"adminWithdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"destinationDomainID\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"resourceID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"deposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"domainID\",\"type\":\"uint8\"},{\"internalType\":\"uint64\",\"name\":\"depositNonce\",\"type\":\"uint64\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"bytes32\",\"name\":\"resourceID\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"revertOnFail\",\"type\":\"bool\"}],\"name\":\"executeProposal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"addresspayable[]\",\"name\":\"addrs\",\"type\":\"address[]\"},{\"internalType\":\"uint256[]\",\"name\":\"amounts\",\"type\":\"uint256[]\"}],\"name\":\"transferFunds\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"startKeygen\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"MPCAddress\",\"type\":\"address\"}],\"name\":\"endKeygen\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"refreshKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"domainID\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"depositNonce\",\"type\":\"uint256\"}],\"name\":\"isProposalExecuted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"
const BridgeBin = "0x60806040523480156200001157600080fd5b5060405162002b5d38038062002b5d833981016040819052620000349162000394565b6000805460ff199081169091556002805490911660ff85161790556200006682620000fc602090811b620015f717901c565b600260016101000a8154816001600160801b0302191690836001600160801b03160217905550620000a2816200015b60201b620016501760201c565b6002805464ffffffffff92909216600160881b0264ffffffffff60881b19909216919091179055620000df6000620000d9620001b4565b620001f7565b620000f3620000ed620001b4565b62000207565b505050620003d3565b6000600160801b8210620001575760405162461bcd60e51b815260206004820152601e60248201527f76616c756520646f6573206e6f742066697420696e203132382062697473000060448201526064015b60405180910390fd5b5090565b6000650100000000008210620001575760405162461bcd60e51b815260206004820152601d60248201527f76616c756520646f6573206e6f742066697420696e203430206269747300000060448201526064016200014e565b60003360143610801590620001e157506001600160a01b03811660009081526006602052604090205460ff165b15620001f2575060131936013560601c5b919050565b6200020382826200025d565b5050565b62000211620002d8565b6000805460ff191660011790556040516001600160a01b03821681527f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a2589060200160405180910390a150565b600082815260016020908152604090912062000284918390620016a762000322821b17901c565b15620002035762000294620001b4565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b60005460ff1615620003205760405162461bcd60e51b815260206004820152601060248201526f14185d5cd8589b194e881c185d5cd95960821b60448201526064016200014e565b565b600062000339836001600160a01b03841662000342565b90505b92915050565b60008181526001830160205260408120546200038b575081546001818101845560008481526020808220909301849055845484825282860190935260409020919091556200033c565b5060006200033c565b600080600060608486031215620003aa57600080fd5b835160ff81168114620003bc57600080fd5b602085015160409095015190969495509392505050565b61277a80620003e36000396000f3fe6080604052600436106102045760003560e01c80639010d07c11610118578063ca15c873116100a0578063d547741f1161006f578063d547741f1461067c578063edc20c3c1461069c578063f5f63b39146106bc578063f8c39e44146106d1578063ffaac0eb1461070157600080fd5b8063ca15c873146105fc578063cb10f2151461061c578063d15ef64e1461063c578063d2e5fae91461065c57600080fd5b80639dd694f4116100e75780639dd694f414610523578063a217fddf1461054f578063bd2a182014610564578063c5b37c2214610584578063c5ec8970146105c157600080fd5b80639010d07c146104a357806391c404ac146104c357806391d14854146104e35780639ae0bf451461050357600080fd5b80634b0b919d1161019b5780635e1fab0f1161016a5780635e1fab0f146104035780636ba6db6b1461042357806380ae1c281461043857806384db809f1461044d5780638c0c26311461048357600080fd5b80634b0b919d146103515780634e0df3f61461039f5780635a1ad87c146103bf5780635c975abb146103df57600080fd5b8063248a9ca3116101d7578063248a9ca3146102c15780632f2ff15d146102f157806336568abe146103115780634603ae381461033157600080fd5b8063059972d21461020957806305e2ca171461024657806308a641041461025b578063202f5b8c146102a1575b600080fd5b34801561021557600080fd5b50600354610229906001600160a01b031681565b6040516001600160a01b0390911681526020015b60405180910390f35b610259610254366004611eda565b610716565b005b34801561026757600080fd5b50610293610276366004611f33565b600760209081526000928352604080842090915290825290205481565b60405190815260200161023d565b3480156102ad57600080fd5b506102596102bc366004611f84565b610910565b3480156102cd57600080fd5b506102936102dc36600461202e565b60009081526001602052604090206002015490565b3480156102fd57600080fd5b5061025961030c36600461205c565b610cf5565b34801561031d57600080fd5b5061025961032c36600461205c565b610d85565b34801561033d57600080fd5b5061025961034c3660046120d0565b610e0f565b34801561035d57600080fd5b5061038761036c36600461212f565b6004602052600090815260409020546001600160401b031681565b6040516001600160401b03909116815260200161023d565b3480156103ab57600080fd5b506102936103ba36600461205c565b610eb3565b3480156103cb57600080fd5b506102596103da366004612162565b610edf565b3480156103eb57600080fd5b5060005460ff165b604051901515815260200161023d565b34801561040f57600080fd5b5061025961041e3660046121cc565b610f95565b34801561042f57600080fd5b50610259611021565b34801561044457600080fd5b506102596110ad565b34801561045957600080fd5b5061022961046836600461202e565b6005602052600090815260409020546001600160a01b031681565b34801561048f57600080fd5b5061025961049e3660046121e9565b6110c7565b3480156104af57600080fd5b506102296104be366004612217565b611133565b3480156104cf57600080fd5b506102596104de36600461202e565b611152565b3480156104ef57600080fd5b506103f36104fe36600461205c565b6111ec565b34801561050f57600080fd5b506103f361051e366004611f33565b611204565b34801561052f57600080fd5b5060025461053d9060ff1681565b60405160ff909116815260200161023d565b34801561055b57600080fd5b50610293600081565b34801561057057600080fd5b5061025961057f3660046122a6565b611253565b34801561059057600080fd5b506002546105a99061010090046001600160801b031681565b6040516001600160801b03909116815260200161023d565b3480156105cd57600080fd5b506002546105e690600160881b900464ffffffffff1681565b60405164ffffffffff909116815260200161023d565b34801561060857600080fd5b5061029361061736600461202e565b611289565b34801561062857600080fd5b50610259610637366004612338565b6112a0565b34801561064857600080fd5b5061025961065736600461237a565b61132a565b34801561066857600080fd5b506102596106773660046121cc565b61135d565b34801561068857600080fd5b5061025961069736600461205c565b611475565b3480156106a857600080fd5b506102596106b73660046123af565b6114f8565b3480156106c857600080fd5b506102596115b1565b3480156106dd57600080fd5b506103f36106ec3660046121cc565b60066020526000908152604090205460ff1681565b34801561070d57600080fd5b506102596115e4565b61071e6116bc565b60025461010090046001600160801b0316341461077b5760405162461bcd60e51b8152602060048201526016602482015275125b98dbdc9c9958dd08199959481cdd5c1c1b1a595960521b60448201526064015b60405180910390fd5b6000838152600560205260409020546001600160a01b0316806107e05760405162461bcd60e51b815260206004820181905260248201527f7265736f757263654944206e6f74206d617070656420746f2068616e646c65726044820152606401610772565b60ff8516600090815260046020526040812080548290610808906001600160401b03166123ef565b91906101000a8154816001600160401b0302191690836001600160401b03160217905590506000610837611702565b60405163b07e54bb60e01b815290915083906000906001600160a01b0383169063b07e54bb90610871908b9087908c908c9060040161243f565b6000604051808303816000875af1158015610890573d6000803e3d6000fd5b505050506040513d6000823e601f3d908101601f191682016040526108b891908101906124a4565b9050826001600160a01b03167f17bc3181e17a9620a479c24e6c606e474ba84fc036877b768926872e8cd0e11f8a8a878b8b876040516108fd96959493929190612546565b60405180910390a2505050505050505050565b6109186116bc565b61092b88886001600160401b0316611204565b1515600114156109925760405162461bcd60e51b815260206004820152602c60248201527f4465706f73697420776974682070726f7669646564206e6f6e636520616c726560448201526b18591e48195e1958dd5d195960a21b6064820152608401610772565b6000610a1089898989896040516020016109b0959493929190612597565b60408051601f1981840301815282825280516020918201207f19457468657265756d205369676e6564204d6573736167653a0a33320000000084830152603c8085019190915282518085039091018152605c909301909152815191012090565b90506000610a548286868080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061174392505050565b6003549091506001600160a01b03808316911614610aad5760405162461bcd60e51b815260206004820152601660248201527524b73b30b634b21036b2b9b9b0b3b29039b4b3b732b960511b6044820152606401610772565b60008681526005602090815260408083205490516001600160a01b039091169291610ade9184918d918d91016125d5565b60408051601f198184030181529190528051602090910120905081610b056101008d612617565b6001600160401b03166001901b600760008f60ff1660ff16815260200190815260200160002060006101008f610b3b919061263d565b6001600160401b031681526020810191909152604001600020805490911790558515610bc85760405163712467f960e11b81526001600160a01b0382169063e248cff290610b91908c908f908f90600401612663565b600060405180830381600087803b158015610bab57600080fd5b505af1158015610bbf573d6000803e3d6000fd5b50505050610c99565b60405163712467f960e11b81526001600160a01b0382169063e248cff290610bf8908c908f908f90600401612663565b600060405180830381600087803b158015610c1257600080fd5b505af1925050508015610c23575060015b610c99573d808015610c51576040519150601f19603f3d011682016040523d82523d6000602084013e610c56565b606091505b507fbd37c1f0d53bb2f33fe4c2104de272fcdeb4d2fef3acdbf1e4ddc3d6833ca37681604051610c86919061267d565b60405180910390a1505050505050610ceb565b6040805160ff8f1681526001600160401b038e1660208201529081018390527f6018c584b8d99bafeda249b2429f5907d830e792222070c1b3a94aa76ee716779060600160405180910390a150505050505b5050505050505050565b600082815260016020526040902060020154610d13906104fe611702565b610d775760405162461bcd60e51b815260206004820152602f60248201527f416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e60448201526e0818591b5a5b881d1bc819dc985b9d608a1b6064820152608401610772565b610d818282611767565b5050565b610d8d611702565b6001600160a01b0316816001600160a01b031614610e055760405162461bcd60e51b815260206004820152602f60248201527f416363657373436f6e74726f6c3a2063616e206f6e6c792072656e6f756e636560448201526e103937b632b9903337b91039b2b63360891b6064820152608401610772565b610d8182826117d0565b610e17611839565b60005b83811015610eac57848482818110610e3457610e34612690565b9050602002016020810190610e4991906121cc565b6001600160a01b03166108fc848484818110610e6757610e67612690565b905060200201359081150290604051600060405180830381858888f19350505050158015610e99573d6000803e3d6000fd5b5080610ea4816126a6565b915050610e1a565b5050505050565b60008281526001602081815260408084206001600160a01b038616855290920190529020545b92915050565b610ee7611839565b6000858152600560205260409081902080546001600160a01b0319166001600160a01b03898116918217909255915163de319d9960e01b81526004810188905290861660248201526001600160e01b03198086166044830152606482018590528316608482015287919063de319d999060a401600060405180830381600087803b158015610f7457600080fd5b505af1158015610f88573d6000803e3d6000fd5b5050505050505050505050565b610f9d611839565b6000610fa7611702565b9050816001600160a01b0316816001600160a01b0316141561100b5760405162461bcd60e51b815260206004820152601760248201527f43616e6e6f742072656e6f756e6365206f6e6573656c660000000000000000006044820152606401610772565b611016600083610cf5565b610d81600082610d85565b611029611839565b6003546001600160a01b0316156110825760405162461bcd60e51b815260206004820152601a60248201527f4d5043206164647265737320697320616c7265616479207365740000000000006044820152606401610772565b6040517f24e723a5c27b62883404028b8dee9965934de6a46828cda2ff63bf9a5e65ce4390600090a1565b6110b5611839565b6110c56110c0611702565b611892565b565b6110cf611839565b6040516307b7ed9960e01b81526001600160a01b0382811660048301528391908216906307b7ed99906024015b600060405180830381600087803b15801561111657600080fd5b505af115801561112a573d6000803e3d6000fd5b50505050505050565b600082815260016020526040812061114b90836118e7565b9392505050565b61115a611839565b60025461010090046001600160801b03168114156111ba5760405162461bcd60e51b815260206004820152601f60248201527f43757272656e742066656520697320657175616c20746f206e657720666565006044820152606401610772565b6111c3816115f7565b600260016101000a8154816001600160801b0302191690836001600160801b0316021790555050565b600082815260016020526040812061114b90836118f3565b6000611212610100836126c1565b60ff84166000908152600760205260408120600190921b9190611237610100866126d5565b8152602001908152602001600020541660001415905092915050565b61125b611839565b60405163025a3c9960e21b815282906001600160a01b03821690630968f264906110fc90859060040161267d565b6000818152600160205260408120610ed990611915565b6112a8611839565b6000828152600560205260409081902080546001600160a01b0319166001600160a01b038681169182179092559151635c7d1b9b60e11b815260048101859052908316602482015284919063b8fa373690604401600060405180830381600087803b15801561131657600080fd5b505af1158015610ceb573d6000803e3d6000fd5b611332611839565b6001600160a01b03919091166000908152600660205260409020805460ff1916911515919091179055565b611365611839565b6001600160a01b0381166113c55760405162461bcd60e51b815260206004820152602160248201527f4d504320616464726573732063616e2774206265207a65726f2d6164647265736044820152607360f81b6064820152608401610772565b6003546001600160a01b03161561141e5760405162461bcd60e51b815260206004820152601c60248201527f4d504320616464726573732063616e27742062652075706461746564000000006044820152606401610772565b600380546001600160a01b0319166001600160a01b038316179055611449611444611702565b61191f565b6040517f4187686ceef7b541a1f224d48d4cded8f2c535e0e58ac0f0514071b1de3dad5790600090a150565b600082815260016020526040902060020154611493906104fe611702565b610e055760405162461bcd60e51b815260206004820152603060248201527f416363657373436f6e74726f6c3a2073656e646572206d75737420626520616e60448201526f2061646d696e20746f207265766f6b6560801b6064820152608401610772565b611500611839565b60ff82166000908152600460205260409020546001600160401b039081169082161161157d5760405162461bcd60e51b815260206004820152602660248201527f446f6573206e6f7420616c6c6f772064656372656d656e7473206f6620746865604482015265206e6f6e636560d01b6064820152608401610772565b60ff919091166000908152600460205260409020805467ffffffffffffffff19166001600160401b03909216919091179055565b6115b9611839565b6040517f034a03238f3c0f2cad22894b0fa8810b6ffcf678a2560e54a7e41b4e9cebd02e90600090a1565b6115ec611839565b6110c5611444611702565b6000600160801b821061164c5760405162461bcd60e51b815260206004820152601e60248201527f76616c756520646f6573206e6f742066697420696e20313238206269747300006044820152606401610772565b5090565b600065010000000000821061164c5760405162461bcd60e51b815260206004820152601d60248201527f76616c756520646f6573206e6f742066697420696e20343020626974730000006044820152606401610772565b600061114b836001600160a01b03841661196a565b60005460ff16156110c55760405162461bcd60e51b815260206004820152601060248201526f14185d5cd8589b194e881c185d5cd95960821b6044820152606401610772565b6000336014361080159061172e57506001600160a01b03811660009081526006602052604090205460ff165b1561173e575060131936013560601c5b919050565b600080600061175285856119b9565b9150915061175f81611a29565b509392505050565b600082815260016020526040902061177f90826116a7565b15610d815761178c611702565b6001600160a01b0316816001600160a01b0316837f2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d60405160405180910390a45050565b60008281526001602052604090206117e89082611be7565b15610d81576117f5611702565b6001600160a01b0316816001600160a01b0316837ff6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b60405160405180910390a45050565b61184660006104fe611702565b6110c55760405162461bcd60e51b815260206004820152601e60248201527f73656e64657220646f65736e277420686176652061646d696e20726f6c6500006044820152606401610772565b61189a6116bc565b6000805460ff191660011790556040516001600160a01b03821681527f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258906020015b60405180910390a150565b600061114b8383611bfc565b6001600160a01b0381166000908152600183016020526040812054151561114b565b6000610ed9825490565b611927611c26565b6000805460ff191690556040516001600160a01b03821681527f5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa906020016118dc565b60008181526001830160205260408120546119b157508154600181810184556000848152602080822090930184905584548482528286019093526040902091909155610ed9565b506000610ed9565b6000808251604114156119f05760208301516040840151606085015160001a6119e487828585611c6f565b94509450505050611a22565b825160401415611a1a5760208301516040840151611a0f868383611d5c565b935093505050611a22565b506000905060025b9250929050565b6000816004811115611a3d57611a3d6126e9565b1415611a465750565b6001816004811115611a5a57611a5a6126e9565b1415611aa85760405162461bcd60e51b815260206004820152601860248201527f45434453413a20696e76616c6964207369676e617475726500000000000000006044820152606401610772565b6002816004811115611abc57611abc6126e9565b1415611b0a5760405162461bcd60e51b815260206004820152601f60248201527f45434453413a20696e76616c6964207369676e6174757265206c656e677468006044820152606401610772565b6003816004811115611b1e57611b1e6126e9565b1415611b775760405162461bcd60e51b815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202773272076616c604482015261756560f01b6064820152608401610772565b6004816004811115611b8b57611b8b6126e9565b1415611be45760405162461bcd60e51b815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202776272076616c604482015261756560f01b6064820152608401610772565b50565b600061114b836001600160a01b038416611d95565b6000826000018281548110611c1357611c13612690565b9060005260206000200154905092915050565b60005460ff166110c55760405162461bcd60e51b815260206004820152601460248201527314185d5cd8589b194e881b9bdd081c185d5cd95960621b6044820152606401610772565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0831115611ca65750600090506003611d53565b8460ff16601b14158015611cbe57508460ff16601c14155b15611ccf5750600090506004611d53565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa158015611d23573d6000803e3d6000fd5b5050604051601f1901519150506001600160a01b038116611d4c57600060019250925050611d53565b9150600090505b94509492505050565b6000806001600160ff1b03831681611d7960ff86901c601b6126ff565b9050611d8787828885611c6f565b935093505050935093915050565b60008181526001830160205260408120548015611e7e576000611db9600183612717565b8554909150600090611dcd90600190612717565b9050818114611e32576000866000018281548110611ded57611ded612690565b9060005260206000200154905080876000018481548110611e1057611e10612690565b6000918252602080832090910192909255918252600188019052604090208390555b8554869080611e4357611e4361272e565b600190038181906000526020600020016000905590558560010160008681526020019081526020016000206000905560019350505050610ed9565b6000915050610ed9565b803560ff8116811461173e57600080fd5b60008083601f840112611eab57600080fd5b5081356001600160401b03811115611ec257600080fd5b602083019150836020828501011115611a2257600080fd5b60008060008060608587031215611ef057600080fd5b611ef985611e88565b93506020850135925060408501356001600160401b03811115611f1b57600080fd5b611f2787828801611e99565b95989497509550505050565b60008060408385031215611f4657600080fd5b611f4f83611e88565b946020939093013593505050565b80356001600160401b038116811461173e57600080fd5b8035801515811461173e57600080fd5b60008060008060008060008060c0898b031215611fa057600080fd5b611fa989611e88565b9750611fb760208a01611f5d565b965060408901356001600160401b0380821115611fd357600080fd5b611fdf8c838d01611e99565b909850965060608b0135955060808b0135915080821115611fff57600080fd5b5061200c8b828c01611e99565b909450925061201f905060a08a01611f74565b90509295985092959890939650565b60006020828403121561204057600080fd5b5035919050565b6001600160a01b0381168114611be457600080fd5b6000806040838503121561206f57600080fd5b82359150602083013561208181612047565b809150509250929050565b60008083601f84011261209e57600080fd5b5081356001600160401b038111156120b557600080fd5b6020830191508360208260051b8501011115611a2257600080fd5b600080600080604085870312156120e657600080fd5b84356001600160401b03808211156120fd57600080fd5b6121098883890161208c565b9096509450602087013591508082111561212257600080fd5b50611f278782880161208c565b60006020828403121561214157600080fd5b61114b82611e88565b80356001600160e01b03198116811461173e57600080fd5b60008060008060008060c0878903121561217b57600080fd5b863561218681612047565b955060208701359450604087013561219d81612047565b93506121ab6060880161214a565b9250608087013591506121c060a0880161214a565b90509295509295509295565b6000602082840312156121de57600080fd5b813561114b81612047565b600080604083850312156121fc57600080fd5b823561220781612047565b9150602083013561208181612047565b6000806040838503121561222a57600080fd5b50508035926020909101359150565b634e487b7160e01b600052604160045260246000fd5b604051601f8201601f191681016001600160401b038111828210171561227757612277612239565b604052919050565b60006001600160401b0382111561229857612298612239565b50601f01601f191660200190565b600080604083850312156122b957600080fd5b82356122c481612047565b915060208301356001600160401b038111156122df57600080fd5b8301601f810185136122f057600080fd5b80356123036122fe8261227f565b61224f565b81815286602083850101111561231857600080fd5b816020840160208301376000602083830101528093505050509250929050565b60008060006060848603121561234d57600080fd5b833561235881612047565b925060208401359150604084013561236f81612047565b809150509250925092565b6000806040838503121561238d57600080fd5b823561239881612047565b91506123a660208401611f74565b90509250929050565b600080604083850312156123c257600080fd5b6123cb83611e88565b91506123a660208401611f5d565b634e487b7160e01b600052601160045260246000fd5b60006001600160401b038083168181141561240c5761240c6123d9565b6001019392505050565b81835281816020850137506000828201602090810191909152601f909101601f19169091010190565b8481526001600160a01b038416602082015260606040820181905260009061246a9083018486612416565b9695505050505050565b60005b8381101561248f578181015183820152602001612477565b8381111561249e576000848401525b50505050565b6000602082840312156124b657600080fd5b81516001600160401b038111156124cc57600080fd5b8201601f810184136124dd57600080fd5b80516124eb6122fe8261227f565b81815285602083850101111561250057600080fd5b612511826020830160208601612474565b95945050505050565b60008151808452612532816020860160208601612474565b601f01601f19169290920160200192915050565b60ff871681528560208201526001600160401b038516604082015260a06060820152600061257860a083018587612416565b828103608084015261258a818561251a565b9998505050505050505050565b60f886901b6001600160f81b031916815260c085901b6001600160c01b03191660018201528284600983013760099201918201526029019392505050565b6bffffffffffffffffffffffff198460601b168152818360148301376000910160140190815292915050565b634e487b7160e01b600052601260045260246000fd5b60006001600160401b038084168061263157612631612601565b92169190910692915050565b60006001600160401b038084168061265757612657612601565b92169190910492915050565b838152604060208201526000612511604083018486612416565b60208152600061114b602083018461251a565b634e487b7160e01b600052603260045260246000fd5b60006000198214156126ba576126ba6123d9565b5060010190565b6000826126d0576126d0612601565b500690565b6000826126e4576126e4612601565b500490565b634e487b7160e01b600052602160045260246000fd5b60008219821115612712576127126123d9565b500190565b600082821015612729576127296123d9565b500390565b634e487b7160e01b600052603160045260246000fdfea2646970667358221220832025445cdf65eccff9c71033cbea8b8e2f7e982a1659a325157a05c78f86ee64736f6c634300080b0033"
