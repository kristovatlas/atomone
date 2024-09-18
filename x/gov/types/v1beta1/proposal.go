package v1beta1

import (
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/cosmos/gogoproto/proto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/atomone-hub/atomone/x/gov/types"
)

// DefaultStartingProposalID is 1
const DefaultStartingProposalID uint64 = 1

// NewProposal creates a new Proposal instance
func NewProposal(content Content, id uint64, submitTime, depositEndTime time.Time) (Proposal, error) {
	msg, ok := content.(proto.Message)
	if !ok {
		return Proposal{}, fmt.Errorf("%T does not implement proto.Message", content)
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return Proposal{}, err
	}

	p := Proposal{
		Content:          any,
		ProposalId:       id,
		Status:           StatusDepositPeriod,
		FinalTallyResult: EmptyTallyResult(),
		TotalDeposit:     sdk.NewCoins(),
		SubmitTime:       submitTime,
		DepositEndTime:   depositEndTime,
	}

	return p, nil
}

// String implements stringer interface
func (p Proposal) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// GetContent returns the proposal Content
func (p Proposal) GetContent() Content {
	content, ok := p.Content.GetCachedValue().(Content)
	if !ok {
		return nil
	}
	return content
}

// ProposalType returns the proposal type
func (p Proposal) ProposalType() string {
	content := p.GetContent()
	if content == nil {
		return ""
	}
	return content.ProposalType()
}

// ProposalRoute returns the proposal route
func (p Proposal) ProposalRoute() string {
	content := p.GetContent()
	if content == nil {
		return ""
	}
	return content.ProposalRoute()
}

// GetTitle gets the proposal's title
func (p Proposal) GetTitle() string {
	content := p.GetContent()
	if content == nil {
		return ""
	}
	return content.GetTitle()
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (p Proposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var content Content
	return unpacker.UnpackAny(p.Content, &content)
}

// Proposals is an array of proposal
type Proposals []Proposal

var _ codectypes.UnpackInterfacesMessage = Proposals{}

// Equal returns true if two slices (order-dependant) of proposals are equal.
func (p Proposals) Equal(other Proposals) bool {
	if len(p) != len(other) {
		return false
	}

	for i, proposal := range p {
		if !proposal.Equal(other[i]) {
			return false
		}
	}

	return true
}

// String implements stringer interface
func (p Proposals) String() string {
	out := "ID - (Status) [Type] Title\n"
	for _, prop := range p {
		out += fmt.Sprintf("%d - (%s) [%s] %s\n",
			prop.ProposalId, prop.Status,
			prop.ProposalType(), prop.GetTitle())
	}
	return strings.TrimSpace(out)
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (p Proposals) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for _, x := range p {
		err := x.UnpackInterfaces(unpacker)
		if err != nil {
			return err
		}
	}
	return nil
}

// ProposalStatusFromString turns a string into a ProposalStatus
func ProposalStatusFromString(str string) (ProposalStatus, error) {
	num, ok := ProposalStatus_value[str]
	if !ok {
		return StatusNil, fmt.Errorf("'%s' is not a valid proposal status", str)
	}
	return ProposalStatus(num), nil
}

// Format implements the fmt.Formatter interface.
func (status ProposalStatus) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		s.Write([]byte(status.String()))
	default:
		// TODO: Do this conversion more directly
		s.Write([]byte(fmt.Sprintf("%v", byte(status))))
	}
}

// Proposal types
const (
	ProposalTypeText string = "Text"

	// Constants pertaining to a Content object
	MaxDescriptionLength int = 10000
	MaxTitleLength       int = 140
)

// Implements Content Interface
var _ Content = &TextProposal{}

// NewTextProposal creates a text proposal Content
func NewTextProposal(title, description string) Content {
	return &TextProposal{title, description}
}

// GetTitle returns the proposal title
func (tp *TextProposal) GetTitle() string { return tp.Title }

// GetDescription returns the proposal description
func (tp *TextProposal) GetDescription() string { return tp.Description }

// ProposalRoute returns the proposal router key
func (tp *TextProposal) ProposalRoute() string { return types.RouterKey }

// ProposalType is "Text"
func (tp *TextProposal) ProposalType() string { return ProposalTypeText }

// ValidateBasic validates the content's title and description of the proposal
func (tp *TextProposal) ValidateBasic() error { return ValidateAbstract(tp) }

// String implements Stringer interface
func (tp TextProposal) String() string {
	out, _ := yaml.Marshal(tp)
	return string(out)
}

// Law and Constitution Amendment Proposal types
const (
	ProposalTypeLaw                   string = "Law"
	ProposalTypeConstitutionAmendment string = "ConstitutionAmendment"
)

// Implements Content Interface
var (
	_ Content = &LawProposal{}
	_ Content = &ConstitutionAmendmentProposal{}
)

// NewLawProposal creates a law proposal Content
func NewLawProposal(title, description string) Content {
	return &LawProposal{title, description}
}

// NewConstitutionAmendmentProposal creates a constitution amendment proposal Content
func NewConstitutionAmendmentProposal(title, description string) Content {
	return &ConstitutionAmendmentProposal{title, description}
}

// GetTitle returns the proposal title
func (lp *LawProposal) GetTitle() string { return lp.Title }

// GetDescription returns the proposal description
func (lp *LawProposal) GetDescription() string { return lp.Description }

// ProposalRoute returns the proposal router key
func (lp *LawProposal) ProposalRoute() string { return types.RouterKey }

// ProposalType is "Law"
func (lp *LawProposal) ProposalType() string { return ProposalTypeLaw }

// ValidateBasic validates the content's title and description of the proposal
func (lp *LawProposal) ValidateBasic() error { return ValidateAbstract(lp) }

// String implements Stringer interface
func (lp LawProposal) String() string {
	out, _ := yaml.Marshal(lp)
	return string(out)
}

// GetTitle returns the proposal title
func (cap *ConstitutionAmendmentProposal) GetTitle() string { return cap.Title }

// GetDescription returns the proposal description
func (cap *ConstitutionAmendmentProposal) GetDescription() string { return cap.Description }

// ProposalRoute returns the proposal router key
func (cap *ConstitutionAmendmentProposal) ProposalRoute() string { return types.RouterKey }

// ProposalType is "ConstitutionAmendment"
func (cap *ConstitutionAmendmentProposal) ProposalType() string {
	return ProposalTypeConstitutionAmendment
}

// ValidateBasic validates the content's title and description of the proposal
func (cap *ConstitutionAmendmentProposal) ValidateBasic() error { return ValidateAbstract(cap) }

// String implements Stringer interface
func (cap ConstitutionAmendmentProposal) String() string {
	out, _ := yaml.Marshal(cap)
	return string(out)
}

// ValidProposalStatus checks if the proposal status is valid
func ValidProposalStatus(status ProposalStatus) bool {
	if status == StatusDepositPeriod ||
		status == StatusVotingPeriod ||
		status == StatusPassed ||
		status == StatusRejected ||
		status == StatusFailed {
		return true
	}
	return false
}

// ValidateAbstract validates a proposal's abstract contents returning an error
// if invalid.
func ValidateAbstract(c Content) error {
	title := c.GetTitle()
	if len(strings.TrimSpace(title)) == 0 {
		return sdkerrors.Wrap(types.ErrInvalidProposalContent, "proposal title cannot be blank") //nolint:staticcheck
	}
	if len(title) > MaxTitleLength {
		return sdkerrors.Wrapf(types.ErrInvalidProposalContent, "proposal title is longer than max length of %d", MaxTitleLength) //nolint:staticcheck
	}

	description := c.GetDescription()
	if len(description) == 0 {
		return sdkerrors.Wrap(types.ErrInvalidProposalContent, "proposal description cannot be blank") //nolint:staticcheck
	}
	if len(description) > MaxDescriptionLength {
		return sdkerrors.Wrapf(types.ErrInvalidProposalContent, "proposal description is longer than max length of %d", MaxDescriptionLength) //nolint:staticcheck
	}

	return nil
}

var validProposalTypes = map[string]struct{}{
	ProposalTypeText:                  {},
	ProposalTypeLaw:                   {},
	ProposalTypeConstitutionAmendment: {},
}

// RegisterProposalType registers a proposal type. It will panic if the type is
// already registered.
func RegisterProposalType(ty string) {
	if _, ok := validProposalTypes[ty]; ok {
		panic(fmt.Sprintf("already registered proposal type: %s", ty))
	}

	validProposalTypes[ty] = struct{}{}
}

// ContentFromProposalType returns a Content object based on the proposal type.
func ContentFromProposalType(title, desc, ty string) (Content, bool) {
	if strings.EqualFold(ty, ProposalTypeText) {
		return NewTextProposal(title, desc), true
	}

	if strings.EqualFold(ty, ProposalTypeLaw) {
		return NewLawProposal(title, desc), true
	}

	if strings.EqualFold(ty, ProposalTypeConstitutionAmendment) {
		return NewConstitutionAmendmentProposal(title, desc), true
	}

	return nil, false
}

// IsValidProposalType returns a boolean determining if the proposal type is
// valid.
//
// NOTE: Modules with their own proposal types must register them.
func IsValidProposalType(ty string) bool {
	_, ok := validProposalTypes[ty]
	return ok
}

// ProposalHandler implements the Handler interface for governance module-based
// proposals (ie. TextProposal ). Since these are
// merely signaling mechanisms at the moment and do not affect state, it
// performs a no-op.
func ProposalHandler(_ sdk.Context, c Content) error {
	switch c.ProposalType() {
	case ProposalTypeText:
		// both proposal types do not change state so this performs a no-op
		return nil
	case ProposalTypeConstitutionAmendment:
		return nil
	case ProposalTypeLaw:
		return nil

	default:
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized gov proposal type: %s", c.ProposalType()) //nolint:staticcheck
	}
}
