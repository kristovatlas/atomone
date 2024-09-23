package e2e

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

/*
testGovSoftwareUpgrade tests passing a gov proposal to upgrade the chain at a given height.
Test Benchmarks:
1. Submission, deposit and vote of message based proposal to upgrade the chain at a height (current height + buffer)
2. Validation that chain halted at upgrade height
3. Teardown & restart chains
4. Reset proposalCounter so subsequent tests have the correct last effective proposal id for chainA
TODO: Perform upgrade in place of chain restart
*/
func (s *IntegrationTestSuite) testGovSoftwareUpgrade() {
	s.Run("software upgrade", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		height := s.getLatestBlockHeight(s.chainA, 0)
		proposalHeight := height + govProposalBlockBuffer
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++

		submitGovFlags := []string{
			"software-upgrade",
			"Upgrade-0",
			"--title='Upgrade V0'",
			"--description='Software Upgrade'",
			"--no-validate",
			fmt.Sprintf("--upgrade-height=%d", proposalHeight),
			"--upgrade-info=my-info",
		}

		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes=0.8,no=0.1,abstain=0.1"}
		s.submitLegacyGovProposal(chainAAPIEndpoint, sender, proposalCounter, upgradetypes.ProposalTypeSoftwareUpgrade, submitGovFlags, depositGovFlags, voteGovFlags, "weighted-vote", true)

		s.verifyChainHaltedAtUpgradeHeight(s.chainA, 0, proposalHeight)
		s.T().Logf("Successfully halted chain at  height %d", proposalHeight)

		s.TearDownSuite()

		s.T().Logf("Restarting containers")
		s.SetupSuite()

		s.Require().Eventually(
			func() bool {
				return s.getLatestBlockHeight(s.chainA, 0) > 0
			},
			30*time.Second,
			time.Second,
		)

		proposalCounter = 0
	})
}

/*
testGovCancelSoftwareUpgrade tests passing a gov proposal that cancels a pending upgrade.
Test Benchmarks:
1. Submission, deposit and vote of message based proposal to upgrade the chain at a height (current height + buffer)
2. Submission, deposit and vote of message based proposal to cancel the pending upgrade
3. Validation that the chain produced blocks past the intended upgrade height
*/
func (s *IntegrationTestSuite) testGovCancelSoftwareUpgrade() {
	s.Run("cancel software upgrade", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()

		sender := senderAddress.String()
		height := s.getLatestBlockHeight(s.chainA, 0)
		proposalHeight := height + 50

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{
			"software-upgrade",
			"Upgrade-1",
			"--title='Upgrade V1'",
			"--description='Software Upgrade'",
			"--no-validate",
			fmt.Sprintf("--upgrade-height=%d", proposalHeight),
		}

		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitLegacyGovProposal(chainAAPIEndpoint, sender, proposalCounter, upgradetypes.ProposalTypeSoftwareUpgrade, submitGovFlags, depositGovFlags, voteGovFlags, "vote", true)

		proposalCounter++
		submitGovFlags = []string{"cancel-software-upgrade", "--title='Upgrade V1'", "--description='Software Upgrade'"}
		depositGovFlags = []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags = []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitLegacyGovProposal(chainAAPIEndpoint, sender, proposalCounter, upgradetypes.ProposalTypeCancelSoftwareUpgrade, submitGovFlags, depositGovFlags, voteGovFlags, "vote", true)

		s.verifyChainPassesUpgradeHeight(s.chainA, 0, proposalHeight)
		s.T().Logf("Successfully canceled upgrade at height %d", proposalHeight)
	})
}

/*
testGovCommunityPoolSpend tests passing a community spend proposal.
Test Benchmarks:
1. Fund Community Pool
2. Submission, deposit and vote of proposal to spend from the community pool to send atoms to a recipient
3. Validation that the recipient balance has increased by proposal amount
*/
func (s *IntegrationTestSuite) testGovCommunityPoolSpend() {
	s.Run("community pool spend", func() {
		s.fundCommunityPool()
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		recipientAddress, _ := s.chainA.validators[1].keyInfo.GetAddress()
		recipient := recipientAddress.String()
		sendAmount := sdk.NewCoin(uatoneDenom, sdk.NewInt(10000000)) // 10uatone
		s.writeGovCommunitySpendProposal(s.chainA, sendAmount, recipient)

		beforeRecipientBalance, err := getSpecificBalance(chainAAPIEndpoint, recipient, uatoneDenom)
		s.Require().NoError(err)

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalCommunitySpendFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "CommunityPoolSpend", submitGovFlags, depositGovFlags, voteGovFlags, "vote")

		s.Require().Eventually(
			func() bool {
				afterRecipientBalance, err := getSpecificBalance(chainAAPIEndpoint, recipient, uatoneDenom)
				s.Require().NoError(err)

				return afterRecipientBalance.Sub(sendAmount).IsEqual(beforeRecipientBalance)
			},
			10*time.Second,
			time.Second,
		)
	})
}

// testGovParamChange tests passing a param change proposal.
func (s *IntegrationTestSuite) testGovParamChange() {
	s.Run("staking param change", func() {
		// check existing params
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		params := s.queryStakingParams(chainAAPIEndpoint)
		oldMaxValidator := params.Params.MaxValidators
		// add 10 to actual max validators
		params.Params.MaxValidators = oldMaxValidator + 10

		s.writeStakingParamChangeProposal(s.chainA, params.Params)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalParamChangeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "cosmos.staking.v1beta1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote")

		newParams := s.queryStakingParams(chainAAPIEndpoint)
		s.Assert().NotEqual(oldMaxValidator, newParams.Params.MaxValidators)
	})
}

func (s *IntegrationTestSuite) submitLegacyGovProposal(chainAAPIEndpoint, sender string, proposalID int, proposalType string, submitFlags []string, depositFlags []string, voteFlags []string, voteCommand string, withDeposit bool) {
	s.T().Logf("Submitting Gov Proposal: %s", proposalType)
	// min deposit of 1000uatone is required in e2e tests, otherwise the gov antehandler causes the proposal to be dropped
	sflags := submitFlags
	if withDeposit {
		sflags = append(sflags, "--deposit=1000uatone")
	}
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "submit-legacy-proposal", sflags, govtypesv1beta1.StatusDepositPeriod)
	s.T().Logf("Depositing Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "deposit", depositFlags, govtypesv1beta1.StatusVotingPeriod)
	s.T().Logf("Voting Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, voteCommand, voteFlags, govtypesv1beta1.StatusPassed)
}

// NOTE: in SDK >= v0.47 the submit-proposal does not have a --deposit flag
// Instead, the deposit is added to the "deposit" field of the proposal JSON (usually stored as a file)
// you can use `atomoned tx gov draft-proposal` to create a proposal file that you can use
// min initial deposit of 100uatone is required in e2e tests, otherwise the proposal would be dropped
func (s *IntegrationTestSuite) submitGovProposal(chainAAPIEndpoint, sender string, proposalID int, proposalType string, submitFlags []string, depositFlags []string, voteFlags []string, voteCommand string) {
	s.T().Logf("Submitting Gov Proposal: %s", proposalType)
	sflags := submitFlags
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "submit-proposal", sflags, govtypesv1beta1.StatusDepositPeriod)
	s.T().Logf("Depositing Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "deposit", depositFlags, govtypesv1beta1.StatusVotingPeriod)
	s.T().Logf("Voting Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, voteCommand, voteFlags, govtypesv1beta1.StatusPassed)
}

func (s *IntegrationTestSuite) verifyChainHaltedAtUpgradeHeight(c *chain, valIdx int, upgradeHeight int64) {
	s.Require().Eventually(
		func() bool {
			currentHeight := s.getLatestBlockHeight(c, valIdx)

			return currentHeight == upgradeHeight
		},
		30*time.Second,
		time.Second,
	)

	counter := 0
	s.Require().Eventually(
		func() bool {
			currentHeight := s.getLatestBlockHeight(c, valIdx)

			if currentHeight > upgradeHeight {
				return false
			}
			if currentHeight == upgradeHeight {
				counter++
			}
			return counter >= 2
		},
		8*time.Second,
		time.Second,
	)
}

func (s *IntegrationTestSuite) verifyChainPassesUpgradeHeight(c *chain, valIdx int, upgradeHeight int64) {
	var currentHeight int64
	s.Require().Eventually(
		func() bool {
			currentHeight = s.getLatestBlockHeight(c, valIdx)
			return currentHeight > upgradeHeight
		},
		30*time.Second,
		time.Second,
		"expected chain height greater than %d: got %d", upgradeHeight, currentHeight,
	)
}

func (s *IntegrationTestSuite) submitGovCommand(chainAAPIEndpoint, sender string, proposalID int, govCommand string, proposalFlags []string, expectedSuccessStatus govtypesv1beta1.ProposalStatus) {
	s.runGovExec(s.chainA, 0, sender, govCommand, proposalFlags, standardFees.String())

	s.Require().Eventually(
		func() bool {
			proposal, err := queryGovProposal(chainAAPIEndpoint, proposalID)
			s.Require().NoError(err)
			return proposal.GetProposal().Status == expectedSuccessStatus
		},
		15*time.Second,
		time.Second,
	)
}

func (s *IntegrationTestSuite) writeStakingParamChangeProposal(c *chain, params stakingtypes.Params) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	template := `
	{
		"messages":[
		  {
			"@type": "/cosmos.staking.v1beta1.MsgUpdateParams",
			"authority": "%s",
			"params": %s
		  }
		],
		"deposit": "100uatone",
		"proposer": "Proposing staking param change",
		"metadata": "",
		"title": "Change in staking params",
		"summary": "summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, cdc.MustMarshalJSON(&params))
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalParamChangeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}
