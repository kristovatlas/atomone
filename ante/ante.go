package ante

import (
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	atomoneerrors "github.com/atomone-hub/atomone/types/errors"
	atomonefeeante "github.com/atomone-hub/atomone/x/globalfee/ante"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	Codec             codec.BinaryCodec
	IBCkeeper         *ibckeeper.Keeper
	GlobalFeeSubspace paramtypes.Subspace
	StakingKeeper     *stakingkeeper.Keeper
	TxFeeChecker      ante.TxFeeChecker
}

func NewAnteHandler(opts HandlerOptions) (sdk.AnteHandler, error) {
	if opts.AccountKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if opts.BankKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if opts.SignModeHandler == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if opts.IBCkeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "IBC keeper is required for AnteHandler")
	}

	if opts.GlobalFeeSubspace.Name() == "" {
		return nil, errorsmod.Wrap(atomoneerrors.ErrNotFound, "globalfee param store is required for AnteHandler")
	}

	if opts.StakingKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrNotFound, "staking param store is required for AnteHandler")
	}

	sigGasConsumer := opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		NewGovVoteDecorator(opts.Codec, opts.StakingKeeper),
		atomonefeeante.NewFeeDecorator(opts.GlobalFeeSubspace, opts.StakingKeeper),
		ante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		ante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(opts.AccountKeeper),
		ante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		ante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		ante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCkeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
