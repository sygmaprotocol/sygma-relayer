// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package events

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Events struct {
	types.EventRecords
	SygmaBridge_Deposit         []Deposit
	SygmaBasicFeeHandler_FeeSet []FeeSet

	SygmaBridge_ProposalExecution       []ProposalExecution
	SygmaBridge_FailedHandlerExecution  []FailedHandlerExecution
	SygmaBridge_Retry                   []Retry
	SygmaBridge_BridgePaused            []BridgePaused
	SygmaBridge_BridgeUnpaused          []BridgeUnpaused
	SygmaBridge_RegisterDestDomain      []RegisterDestDomain
	SygmaBridge_UnRegisterDestDomain    []UnregisterDestDomain
	SygmaFeeHandlerRouter_FeeHandlerSet []FeeHandlerSet

	// Substrate default events
	PhragmenElection_CandidateSlashed  []types.EventElectionsCandidateSlashed
	PhragmenElection_ElectionError     []types.EventElectionsElectionError
	PhragmenElection_EmptyTerm         []types.EventElectionsEmptyTerm
	PhragmenElection_MemberKicked      []types.EventElectionsMemberKicked
	PhragmenElection_NewTerm           []types.EventElectionsNewTerm
	PhragmenElection_Renounced         []types.EventElectionsRenounced
	PhragmenElection_SeatHolderSlashed []types.EventElectionsSeatHolderSlashed
}

type Deposit struct {
	Phase        types.Phase
	DestDomainID types.U8
	ResourceID   types.Bytes32
	DepositNonce types.U64
	Sender       types.AccountID
	TransferType [1]byte
	CallData     []byte
	Handler      [1]byte
	Topics       []types.Hash
}

type FeeSet struct {
	Phase    types.Phase
	DomainID types.U8
	Asset    types.AssetID
	Amount   types.U128
	Topics   []types.Hash
}

type ProposalExecution struct {
	Phase          types.Phase
	OriginDomainID types.U8
	DepositNonce   types.U64
	DataHash       types.Bytes32
	Topics         []types.Hash
}

type FailedHandlerExecution struct {
	Phase          types.Phase
	Error          []byte
	OriginDomainID types.U8
	DepositNonce   types.U64
	Topics         []types.Hash
}

type Retry struct {
	Phase                types.Phase
	DepositOnBlockHeight types.U128
	DestDomainID         types.U8
	Sender               types.AccountID
	Topics               []types.Hash
}

type BridgePaused struct {
	Phase        types.Phase
	DestDomainID types.U8
	Topics       []types.Hash
}

type BridgeUnpaused struct {
	Phase        types.Phase
	DestDomainID types.U8
	Topics       []types.Hash
}

type RegisterDestDomain struct {
	Phase    types.Phase
	Sender   types.AccountID
	DomainID types.U8
	ChainID  types.U256
	Topics   []types.Hash
}

type UnregisterDestDomain struct {
	Phase    types.Phase
	Sender   types.AccountID
	DomainID types.U8
	ChainID  types.U256
	Topics   []types.Hash
}

type FeeHandlerSet struct {
	Phase       types.Phase
	DomainID    types.U8
	Asset       types.AssetID
	HandlerType [1]byte
	Topics      []types.Hash
}



// rmrk events
CollectionCreated {
	issuer: T::AccountId,
	collection_id: T::CollectionId,
},
NftMinted {
	owner: AccountIdOrCollectionNftTuple<T::AccountId, T::CollectionId, T::ItemId>,
	collection_id: T::CollectionId,
	nft_id: T::ItemId,
},
NFTBurned {
	owner: T::AccountId,
	collection_id: T::CollectionId,
	nft_id: T::ItemId,
},
CollectionDestroyed {
	issuer: T::AccountId,
	collection_id: T::CollectionId,
},
NFTSent {
	sender: T::AccountId,
	recipient: AccountIdOrCollectionNftTuple<T::AccountId, T::CollectionId, T::ItemId>,
	collection_id: T::CollectionId,
	nft_id: T::ItemId,
	approval_required: bool,
},
NFTAccepted {
	sender: T::AccountId,
	recipient: AccountIdOrCollectionNftTuple<T::AccountId, T::CollectionId, T::ItemId>,
	collection_id: T::CollectionId,
	nft_id: T::ItemId,
},
NFTRejected {
	sender: T::AccountId,
	collection_id: T::CollectionId,
	nft_id: T::ItemId,
},
IssuerChanged {
	old_issuer: T::AccountId,
	new_issuer: T::AccountId,
	collection_id: T::CollectionId,
},
PropertySet {
	collection_id: T::CollectionId,
	maybe_nft_id: Option<T::ItemId>,
	key: KeyLimitOf<T>,
	value: ValueLimitOf<T>,
},
PropertyRemoved {
	collection_id: T::CollectionId,
	maybe_nft_id: Option<T::ItemId>,
	key: KeyLimitOf<T>,
},
PropertiesRemoved {
	collection_id: T::CollectionId,
	maybe_nft_id: Option<T::ItemId>,
},
CollectionLocked {
	issuer: T::AccountId,
	collection_id: T::CollectionId,
},
ResourceAdded {
	nft_id: T::ItemId,
	resource_id: ResourceId,
	collection_id: T::CollectionId,
},
ResourceReplaced {
	nft_id: T::ItemId,
	resource_id: ResourceId,
	collection_id: T::CollectionId,
},
ResourceAccepted {
	nft_id: T::ItemId,
	resource_id: ResourceId,
	collection_id: T::CollectionId,
},
ResourceRemoval {
	nft_id: T::ItemId,
	resource_id: ResourceId,
	collection_id: T::CollectionId,
},
ResourceRemovalAccepted {
	nft_id: T::ItemId,
	resource_id: ResourceId,
	collection_id: T::CollectionId,
},
PrioritySet {
	collection_id: T::CollectionId,
	nft_id: T::ItemId,
}


		/// Asset is registerd.
		AssetRegistered {
			asset_id: <T as pallet_assets::Config>::AssetId,
			location: MultiLocation,
		},
		/// Asset is unregisterd.
		AssetUnregistered {
			asset_id: <T as pallet_assets::Config>::AssetId,
			location: MultiLocation,
		},
		/// Asset enabled chainbridge.
		ChainbridgeEnabled {
			asset_id: <T as pallet_assets::Config>::AssetId,
			chain_id: u8,
			resource_id: [u8; 32],
		},
		/// Asset disabled chainbridge.
		ChainbridgeDisabled {
			asset_id: <T as pallet_assets::Config>::AssetId,
			chain_id: u8,
			resource_id: [u8; 32],
		},
		/// Asset enabled sygmabridge.
		SygmabridgeEnabled {
			asset_id: <T as pallet_assets::Config>::AssetId,
			domain_id: DomainID,
			resource_id: [u8; 32],
		},
		/// Asset disabled sygmabridge.
		SygmabridgeDisabled {
			asset_id: <T as pallet_assets::Config>::AssetId,
			domain_id: DomainID,
			resource_id: [u8; 32],
		},
		/// Force mint asset to an certain account.
		ForceMinted {
			asset_id: <T as pallet_assets::Config>::AssetId,
			beneficiary: T::AccountId,
			amount: <T as pallet_assets::Config>::Balance,
		},
		/// Force burn asset from an certain account.
		ForceBurnt {
			asset_id: <T as pallet_assets::Config>::AssetId,
			who: T::AccountId,
			amount: <T as pallet_assets::Config>::Balance,
		},





		ContractDepositChanged {
			cluster: Option<ContractClusterId>,
			contract: ContractId,
			deposit: BalanceOf<T>,
		},
		UserStakeChanged {
			cluster: Option<ContractClusterId>,
			account: T::AccountId,
			contract: ContractId,
			stake: BalanceOf<T>,
		},




		ClusterCreated {
			cluster: ContractClusterId,
			system_contract: ContractId,
		},
		ClusterPubkeyAvailable {
			cluster: ContractClusterId,
			pubkey: ClusterPublicKey,
		},
		ClusterDeployed {
			cluster: ContractClusterId,
			pubkey: ClusterPublicKey,
			worker: WorkerPublicKey,
		},
		ClusterDeploymentFailed {
			cluster: ContractClusterId,
			worker: WorkerPublicKey,
		},
		Instantiating {
			contract: ContractId,
			cluster: ContractClusterId,
			deployer: T::AccountId,
		},
		ContractPubkeyAvailable {
			contract: ContractId,
			cluster: ContractClusterId,
			pubkey: ContractPublicKey,
		},
		Instantiated {
			contract: ContractId,
			cluster: ContractClusterId,
			deployer: H256,
		},
		ClusterDestroyed {
			cluster: ContractClusterId,
		},
		Transfered {
			cluster: ContractClusterId,
			account: H256,
			amount: BalanceOf<T>,
		},




				/// A new Gatekeeper is enabled on the blockchain
				GatekeeperAdded {
					pubkey: WorkerPublicKey,
				},
				GatekeeperRemoved {
					pubkey: WorkerPublicKey,
				},
				WorkerAdded {
					pubkey: WorkerPublicKey,
					attestation_provider: Option<AttestationProvider>,
					confidence_level: u8,
				},
				WorkerUpdated {
					pubkey: WorkerPublicKey,
					attestation_provider: Option<AttestationProvider>,
					confidence_level: u8,
				},
				MasterKeyRotated {
					rotation_id: u64,
					master_pubkey: MasterPublicKey,
				},
				MasterKeyRotationFailed {
					rotation_lock: Option<u64>,
					gatekeeper_rotation_id: u64,
				},
				InitialScoreSet {
					pubkey: WorkerPublicKey,
					init_score: u32,
				},
				MinimumPRuntimeVersionChangedTo(u32, u32, u32),
				PRuntimeConsensusVersionChangedTo(u32),



						/// A Nft is created to contain pool shares
		NftCreated {
			pid: u64,
			cid: CollectionId,
			nft_id: NftId,
			owner: T::AccountId,
			shares: BalanceOf<T>,
		},
		/// A withdrawal request is inserted to a queue
		///
		/// Affected states:
		/// - a new item is inserted to or an old item is being replaced by the new item in the
		///   withdraw queue in [`Pools`]
		WithdrawalQueued {
			pid: u64,
			user: T::AccountId,
			shares: BalanceOf<T>,
			nft_id: NftId,
			as_vault: Option<u64>,
		},
		/// Some stake was withdrawn from a pool
		///
		/// The lock in [`Balances`](pallet_balances::pallet::Pallet) is updated to release the
		/// locked stake.
		///
		/// Affected states:
		/// - the stake related fields in [`Pools`]
		/// - the user staking asset account
		Withdrawal {
			pid: u64,
			user: T::AccountId,
			amount: BalanceOf<T>,
			shares: BalanceOf<T>,
		},
		/// A pool contribution whitelist is added
		///
		/// - lazy operated when the first staker is added to the whitelist
		PoolWhitelistCreated { pid: u64 },

		/// The pool contribution whitelist is deleted
		///
		/// - lazy operated when the last staker is removed from the whitelist
		PoolWhitelistDeleted { pid: u64 },

		/// A staker is added to the pool contribution whitelist
		PoolWhitelistStakerAdded { pid: u64, staker: T::AccountId },

		/// A staker is removed from the pool contribution whitelist
		PoolWhitelistStakerRemoved { pid: u64, staker: T::AccountId },




				/// Cool down expiration changed (in sec).
		///
		/// Indicates a change in [`CoolDownPeriod`].
		CoolDownExpirationChanged { period: u64 },
		/// A worker starts computing.
		///
		/// Affected states:
		/// - the worker info at [`Sessions`] is updated with `WorkerIdle` state
		/// - [`NextSessionId`] for the session is incremented
		/// - [`Stakes`] for the session is updated
		/// - [`OnlineWorkers`] is incremented
		WorkerStarted {
			session: T::AccountId,
			init_v: u128,
			init_p: u32,
		},
		/// Worker stops computing.
		///
		/// Affected states:
		/// - the worker info at [`Sessions`] is updated with `WorkerCoolingDown` state
		/// - [`OnlineWorkers`] is decremented
		WorkerStopped { session: T::AccountId },
		/// Worker is reclaimed, with its slash settled.
		WorkerReclaimed {
			session: T::AccountId,
			original_stake: BalanceOf<T>,
			slashed: BalanceOf<T>,
		},
		/// Worker & session are bounded.
		///
		/// Affected states:
		/// - [`SessionBindings`] for the session account is pointed to the worker
		/// - [`WorkerBindings`] for the worker is pointed to the session account
		/// - the worker info at [`Sessions`] is updated with `Ready` state
		SessionBound {
			session: T::AccountId,
			worker: WorkerPublicKey,
		},
		/// Worker & worker are unbound.
		///
		/// Affected states:
		/// - [`SessionBindings`] for the session account is removed
		/// - [`WorkerBindings`] for the worker is removed
		SessionUnbound {
			session: T::AccountId,
			worker: WorkerPublicKey,
		},
		/// Worker enters unresponsive state.
		///
		/// Affected states:
		/// - the worker info at [`Sessions`] is updated from `WorkerIdle` to `WorkerUnresponsive`
		WorkerEnterUnresponsive { session: T::AccountId },
		/// Worker returns to responsive state.
		///
		/// Affected states:
		/// - the worker info at [`Sessions`] is updated from `WorkerUnresponsive` to `WorkerIdle`
		WorkerExitUnresponsive { session: T::AccountId },
		/// Worker settled successfully.
		///
		/// It results in the v in [`Sessions`] being updated. It also indicates the downstream
		/// stake pool has received the computing reward (payout), and the treasury has received the
		/// tax.
		SessionSettled {
			session: T::AccountId,
			v_bits: u128,
			payout_bits: u128,
		},
		/// Some internal error happened when settling a worker's ledger.
		InternalErrorWorkerSettleFailed { worker: WorkerPublicKey },
		/// Block subsidy halved by 25%.
		///
		/// This event will be followed by a [`TokenomicParametersChanged`](#variant.TokenomicParametersChanged)
		/// event indicating the change of the block subsidy budget in the parameter.
		SubsidyBudgetHalved,
		/// Some internal error happened when trying to halve the subsidy
		InternalErrorWrongHalvingConfigured,
		/// Tokenomic parameter changed.
		///
		/// Affected states:
		/// - [`TokenomicParameters`] is updated.
		TokenomicParametersChanged,
		/// A session settlement was dropped because the on-chain version is more up-to-date.
		///
		/// This is a temporary walk-around of the computing staking design. Will be fixed by
		/// StakePool v2.
		SessionSettlementDropped {
			session: T::AccountId,
			v: u128,
			payout: u128,
		},
		/// Benchmark Updated
		BenchmarkUpdated {
			session: T::AccountId,
			p_instant: u32,
		},




				/// A stake pool is created by `owner`
		///
		/// Affected states:
		/// - a new entry in [`Pools`] with the pid
		PoolCreated {
			owner: T::AccountId,
			pid: u64,
			cid: CollectionId,
			pool_account_id: T::AccountId,
		},

		/// The commission of a pool is updated
		///
		/// The commission ratio is represented by an integer. The real value is
		/// `commission / 1_000_000u32`.
		///
		/// Affected states:
		/// - the `payout_commission` field in [`Pools`] is updated
		PoolCommissionSet { pid: u64, commission: u32 },

		/// The stake capacity of the pool is updated
		///
		/// Affected states:
		/// - the `cap` field in [`Pools`] is updated
		PoolCapacitySet { pid: u64, cap: BalanceOf<T> },

		/// A worker is added to the pool
		///
		/// Affected states:
		/// - the `worker` is added to the vector `workers` in [`Pools`]
		/// - the worker in the [`WorkerAssignments`] is pointed to `pid`
		/// - the worker-session binding is updated in `computation` pallet ([`WorkerBindings`](computation::pallet::WorkerBindings),
		///   [`SessionBindings`](computation::pallet::SessionBindings))
		PoolWorkerAdded {
			pid: u64,
			worker: WorkerPublicKey,
			session: T::AccountId,
		},

		/// Someone contributed to a pool
		///
		/// Affected states:
		/// - the stake related fields in [`Pools`]
		/// - the user W-PHA balance reduced
		/// - the user recive ad share NFT once contribution succeeded
		/// - when there was any request in the withdraw queue, the action may trigger withdrawals
		///   ([`Withdrawal`](#variant.Withdrawal) event)
		Contribution {
			pid: u64,
			user: T::AccountId,
			amount: BalanceOf<T>,
			shares: BalanceOf<T>,
			as_vault: Option<u64>,
		},

		/// Owner rewards were withdrawn by pool owner
		///
		/// Affected states:
		/// - the stake related fields in [`Pools`]
		/// - the owner asset account
		OwnerRewardsWithdrawn {
			pid: u64,
			user: T::AccountId,
			amount: BalanceOf<T>,
		},

		/// The pool received a slash event from one of its workers (currently disabled)
		///
		/// The slash is accured to the pending slash accumulator.
		PoolSlashed { pid: u64, amount: BalanceOf<T> },

		/// Some slash is actually settled to a contributor (currently disabled)
		SlashSettled {
			pid: u64,
			user: T::AccountId,
			amount: BalanceOf<T>,
		},

		/// Some reward is dismissed because the worker is no longer bound to a pool
		///
		/// There's no affected state.
		RewardDismissedNotInPool {
			worker: WorkerPublicKey,
			amount: BalanceOf<T>,
		},

		/// Some reward is dismissed because the pool doesn't have any share
		///
		/// There's no affected state.
		RewardDismissedNoShare { pid: u64, amount: BalanceOf<T> },

		/// Some reward is dismissed because the amount is too tiny (dust)
		///
		/// There's no affected state.
		RewardDismissedDust { pid: u64, amount: BalanceOf<T> },

		/// A worker is removed from a pool.
		///
		/// Affected states:
		/// - the worker item in [`WorkerAssignments`] is removed
		/// - the worker is removed from the [`Pools`] item
		PoolWorkerRemoved { pid: u64, worker: WorkerPublicKey },

		/// A worker is reclaimed from the pool
		WorkerReclaimed { pid: u64, worker: WorkerPublicKey },

		/// The amount of reward that distributed to owner and stakers
		RewardReceived {
			pid: u64,
			to_owner: BalanceOf<T>,
			to_stakers: BalanceOf<T>,
		},

		/// The amount of stakes for a worker to start computing
		WorkingStarted {
			pid: u64,
			worker: WorkerPublicKey,
			amount: BalanceOf<T>,
		},



				/// A vault is created by `owner`
		///
		/// Affected states:
		/// - a new entry in [`Pools`] with the pid
		PoolCreated {
			owner: T::AccountId,
			pid: u64,
			cid: CollectionId,
			pool_account_id: T::AccountId,
		},

		/// The commission of a vault is updated
		///
		/// The commission ratio is represented by an integer. The real value is
		/// `commission / 1_000_000u32`.
		///
		/// Affected states:
		/// - the `commission` field in [`Pools`] is updated
		VaultCommissionSet { pid: u64, commission: u32 },

		/// Owner shares is claimed by pool owner
		/// Affected states:
		/// - the shares related fields in [`Pools`]
		/// - the nft related storages in rmrk and pallet unique
		OwnerSharesClaimed {
			pid: u64,
			user: T::AccountId,
			shares: BalanceOf<T>,
		},

		/// Additional owner shares are mint into the pool
		///
		/// Affected states:
		/// - the shares related fields in [`Pools`]
		/// - last_share_price_checkpoint in [`Pools`]
		OwnerSharesGained {
			pid: u64,
			shares: BalanceOf<T>,
			checkout_price: BalanceOf<T>,
		},

		/// Someone contributed to a vault
		///
		/// Affected states:
		/// - the stake related fields in [`Pools`]
		/// - the user W-PHA balance reduced
		/// - the user recive ad share NFT once contribution succeeded
		/// - when there was any request in the withdraw queue, the action may trigger withdrawals
		///   ([`Withdrawal`](#variant.Withdrawal) event)
		Contribution {
			pid: u64,
			user: T::AccountId,
			amount: BalanceOf<T>,
			shares: BalanceOf<T>,
		},



				/// Some dust stake is removed
		///
		/// Triggered when the remaining stake of a user is too small after withdrawal or slash.
		///
		/// Affected states:
		/// - the balance of the locking ledger of the contributor at [`StakeLedger`] is set to 0
		/// - the user's dust stake is moved to treasury
		DustRemoved {
			user: T::AccountId,
			amount: BalanceOf<T>,
		},
		Wrapped {
			user: T::AccountId,
			amount: BalanceOf<T>,
		},
		Unwrapped {
			user: T::AccountId,
			amount: BalanceOf<T>,
		},
		Voted {
			user: T::AccountId,
			vote_id: ReferendumIndex,
			aye_amount: BalanceOf<T>,
			nay_amount: BalanceOf<T>,
		},



				/// CanStartIncubation status changed and set official hatch time.
				CanStartIncubationStatusChanged {
					status: bool,
					start_time: u64,
					official_hatch_time: u64,
				},
				/// Origin of Shell owner has initiated the incubation sequence.
				StartedIncubation {
					collection_id: CollectionId,
					nft_id: NftId,
					owner: T::AccountId,
					start_time: u64,
					hatch_time: u64,
				},
				/// Origin of Shell received food from an account.
				OriginOfShellReceivedFood {
					collection_id: CollectionId,
					nft_id: NftId,
					sender: T::AccountId,
					era: EraId,
				},
				/// Origin of Shell updated chosen parts.
				OriginOfShellChosenPartsUpdated {
					collection_id: CollectionId,
					nft_id: NftId,
					old_chosen_parts: Option<ShellPartsOf<T>>,
					new_chosen_parts: ShellPartsOf<T>,
				},
				/// Shell Collection ID is set.
				ShellCollectionIdSet { collection_id: CollectionId },
				/// Shell Parts Collection ID is set.
				ShellPartsCollectionIdSet { collection_id: CollectionId },
				/// Shell Part minted.
				ShellPartMinted {
					shell_parts_collection_id: CollectionId,
					shell_part_nft_id: NftId,
					parent_shell_collection_id: CollectionId,
					parent_shell_nft_id: NftId,
					owner: T::AccountId,
				},
				/// Shell has been awakened from an origin_of_shell being hatched and burned.
				ShellAwakened {
					shell_collection_id: CollectionId,
					shell_nft_id: NftId,
					rarity: RarityType,
					career: CareerType,
					race: RaceType,
					generation_id: GenerationId,
					origin_of_shell_collection_id: CollectionId,
					origin_of_shell_nft_id: NftId,
					owner: T::AccountId,
				},



						/// Marketplace owner is set.
		MarketplaceOwnerSet {
			old_marketplace_owner: Option<T::AccountId>,
			new_marketplace_owner: T::AccountId,
		},
		/// RoyaltyInfo updated for a NFT.
		RoyaltyInfoUpdated {
			collection_id: T::CollectionId,
			nft_id: T::ItemId,
			old_royalty_info: Option<RoyaltyInfoOf<T>>,
			new_royalty_info: RoyaltyInfoOf<T>,
		},



				/// Phala World clock zero day started
				WorldClockStarted { start_time: u64 },
				/// Start of a new era
				NewEra { time: u64, era: u64 },
				/// Spirit has been claimed
				SpiritClaimed {
					owner: T::AccountId,
					collection_id: CollectionId,
					nft_id: NftId,
				},
				/// A chance to get an Origin of Shell through preorder
				OriginOfShellPreordered {
					owner: T::AccountId,
					preorder_id: PreorderId,
					race: RaceType,
					career: CareerType,
				},
				/// Origin of Shell minted from the preorder
				OriginOfShellMinted {
					rarity_type: RarityType,
					collection_id: CollectionId,
					nft_id: NftId,
					owner: T::AccountId,
					race: RaceType,
					career: CareerType,
					generation_id: GenerationId,
				},
				/// Spirit collection id was set
				SpiritCollectionIdSet { collection_id: CollectionId },
				/// Origin of Shell collection id was set
				OriginOfShellCollectionIdSet { collection_id: CollectionId },
				/// Origin of Shell inventory updated
				OriginOfShellInventoryUpdated { rarity_type: RarityType },
				/// Spirit Claims status has changed
				ClaimSpiritStatusChanged { status: bool },
				/// Purchase Rare Origin of Shells status has changed
				PurchaseRareOriginOfShellsStatusChanged { status: bool },
				/// Purchase Prime Origin of Shells status changed
				PurchasePrimeOriginOfShellsStatusChanged { status: bool },
				/// Preorder Origin of Shells status has changed
				PreorderOriginOfShellsStatusChanged { status: bool },
				/// Chosen preorder was minted to owner
				ChosenPreorderMinted {
					preorder_id: PreorderId,
					owner: T::AccountId,
					nft_id: NftId,
				},
				/// Not chosen preorder was refunded to owner
				NotChosenPreorderRefunded {
					preorder_id: PreorderId,
					owner: T::AccountId,
				},
				/// Last Day of Sale status has changed
				LastDayOfSaleStatusChanged { status: bool },
				OverlordChanged {
					old_overlord: Option<T::AccountId>,
					new_overlord: T::AccountId,
				},
				/// Origin of Shells Inventory was set
				OriginOfShellsInventoryWasSet { status: bool },
				/// Gift a Origin of Shell for giveaway or reserved NFT to owner
				OriginOfShellGiftedToOwner {
					owner: T::AccountId,
					nft_sale_type: NftSaleType,
				},
				/// Spirits Metadata was set
				SpiritsMetadataSet {
					spirits_metadata: BoundedVec<u8, T::StringLimit>,
				},
				/// Origin of Shells Metadata was set
				OriginOfShellsMetadataSet {
					origin_of_shells_metadata: Vec<(RaceType, BoundedVec<u8, T::StringLimit>)>,
				},
				/// Payee changed to new account
				PayeeChanged {
					old_payee: Option<T::AccountId>,
					new_payee: T::AccountId,
				},
				/// Signer changed to new account
				SignerChanged {
					old_signer: Option<T::AccountId>,
					new_signer: T::AccountId,
				},



		/// Vote threshold has changed (new_threshold)
		RelayerThresholdChanged(u32),
		/// Chain now available for transfers (chain_id)
		ChainWhitelisted(BridgeChainId),
		/// Relayer added to set
		RelayerAdded(T::AccountId),
		/// Relayer removed from set
		RelayerRemoved(T::AccountId),
		/// FungibleTransfer is for relaying fungibles (dest_id, nonce, resource_id, amount, recipient)
		FungibleTransfer(BridgeChainId, DepositNonce, ResourceId, U256, Vec<u8>),
		/// NonFungibleTransfer is for relaying NFTs (dest_id, nonce, resource_id, token_id, recipient, metadata)
		NonFungibleTransfer(
			BridgeChainId,
			DepositNonce,
			ResourceId,
			Vec<u8>,
			Vec<u8>,
			Vec<u8>,
		),
		/// GenericTransfer is for a generic data payload (dest_id, nonce, resource_id, metadata)
		GenericTransfer(BridgeChainId, DepositNonce, ResourceId, Vec<u8>),
		/// Vote submitted in favour of proposal
		VoteFor(BridgeChainId, DepositNonce, T::AccountId),
		/// Vot submitted against proposal
		VoteAgainst(BridgeChainId, DepositNonce, T::AccountId),
		/// Voting successful for a proposal
		ProposalApproved(BridgeChainId, DepositNonce),
		/// Voting rejected a proposal
		ProposalRejected(BridgeChainId, DepositNonce),
		/// Execution of call succeeded
		ProposalSucceeded(BridgeChainId, DepositNonce),
		/// Execution of call failed
		ProposalFailed(BridgeChainId, DepositNonce),
		FeeUpdated {
			dest_id: BridgeChainId,
			fee: u128,
		},

//sygmawrapper
		AssetTransfered {
			asset: MultiAsset,
			origin: MultiLocation,
			dest: MultiLocation,
		},

//xcmbridge
		AssetTransfered {
			asset: MultiAsset,
			origin: MultiLocation,
			dest: MultiLocation,
		},


				/// Assets being withdrawn from somewhere.
				Withdrawn {
					what: MultiAsset,
					who: MultiLocation,
					memo: Vec<u8>,
				},
				/// Assets being deposited to somewhere.
				Deposited {
					what: MultiAsset,
					who: MultiLocation,
					memo: Vec<u8>,
				},
				/// Assets being forwarded to somewhere.
				Forwarded {
					what: MultiAsset,
					who: MultiLocation,
					memo: Vec<u8>,
				},