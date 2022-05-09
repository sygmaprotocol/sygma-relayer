package feeHandler

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
	"testing"
)

var (
	validAddr   = "0xd606A00c1A39dA53EA7Bb3Ab570BBE40b156EB66"
	invalidAddr = "0xd606A00c1A39dA53EA7Bb3Ab570BBE40b156EXYZ"
)

type FeeHandlerTestSuite struct {
	suite.Suite
}

func TestFeeHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(FeeHandlerTestSuite))
}

func (f *FeeHandlerTestSuite) SetupSuite() {
}
func (f *FeeHandlerTestSuite) TearDownSuite() {}

func (f *FeeHandlerTestSuite) SetupTest() {}

func (f *FeeHandlerTestSuite) TearDownTest() {}

func (f *FeeHandlerTestSuite) TestValidateChangeFeeValidAddress() {
	cmd := new(cobra.Command)
	BindChangeFeeFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(validAddr)
	f.Nil(err)

	err = ValidateChangeFeeFlags(
		cmd,
		[]string{},
	)
	f.Nil(err)
}

func (f *FeeHandlerTestSuite) TestValidateChangeFeeInvalidAddress() {
	cmd := new(cobra.Command)
	BindChangeFeeFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(invalidAddr)
	f.Nil(err)

	err = ValidateChangeFeeFlags(
		cmd,
		[]string{},
	)
	f.NotNil(err)
}

func (f *FeeHandlerTestSuite) TestValidateSetFeeOracleValidAddress() {
	cmd := new(cobra.Command)
	BindSetFeeOracleFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(validAddr)
	f.Nil(err)
	err = cmd.Flag("fee-oracle").Value.Set(validAddr)
	f.Nil(err)

	err = ValidateSetFeeOracleFlags(
		cmd,
		[]string{},
	)
	f.Nil(err)
}

func (f *FeeHandlerTestSuite) TestValidateSetFeeOracleInvalidAddress() {
	cmd := new(cobra.Command)
	BindSetFeeOracleFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(invalidAddr)
	f.Nil(err)
	err = cmd.Flag("fee-oracle").Value.Set(invalidAddr)
	f.Nil(err)

	err = ValidateSetFeeOracleFlags(
		cmd,
		[]string{},
	)
	f.NotNil(err)
}

func (f *FeeHandlerTestSuite) TestValidateSetFeePropertiesValidFlags() {
	cmd := new(cobra.Command)
	BindSetFeePropertiesFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(validAddr)
	f.Nil(err)
	err = cmd.Flag("gas-used").Value.Set("1000000000")
	f.Nil(err)

	err = ValidateSetFeePropertiesFlags(
		cmd,
		[]string{},
	)
	f.Nil(err)
}

func (f *FeeHandlerTestSuite) TestValidateSetFeePropertiesInvalidFlags() {
	cmd := new(cobra.Command)
	BindSetFeePropertiesFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(invalidAddr)
	f.Nil(err)
	err = cmd.Flag("gas-used").Value.Set("0")
	f.Nil(err)

	err = ValidateSetFeePropertiesFlags(
		cmd,
		[]string{},
	)
	f.NotNil(err)
}

func (f *FeeHandlerTestSuite) TestValidateDistributeFeeValidFlagsBasicFeeHandler() {
	cmd := new(cobra.Command)
	BindDistributeFeeFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(validAddr)
	f.Nil(err)
	err = cmd.Flag("distribution-array").Value.Set(fmt.Sprintf("%s,10", validAddr))
	f.Nil(err)

	err = ValidateDistributeFeeFlags(
		cmd,
		[]string{},
	)
	f.Nil(err)
}

func (f *FeeHandlerTestSuite) TestValidateDistributeFeeInvalidFlagsBasicFeeHandler() {
	cmd := new(cobra.Command)
	BindDistributeFeeFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(invalidAddr)
	f.Nil(err)
	err = cmd.Flag("distribution-array").Value.Set(fmt.Sprintf("%s,10a", invalidAddr))
	f.Nil(err)

	err = ValidateDistributeFeeFlags(
		cmd,
		[]string{},
	)
	f.NotNil(err)
}

func (f *FeeHandlerTestSuite) TestValidateDistributeFeeValidFlagsFeeHandlerWithFeeOracle() {
	cmd := new(cobra.Command)
	BindDistributeFeeFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(validAddr)
	f.Nil(err)
	err = cmd.Flag("distribution-array").Value.Set(fmt.Sprintf("%s,10", validAddr))
	f.Nil(err)
	err = cmd.Flag("fee-handler-with-oracle").Value.Set("true")
	f.Nil(err)
	err = cmd.Flag("resource-id").Value.Set(validAddr)
	f.Nil(err)
	err = cmd.Flag("decimals").Value.Set("18")
	f.Nil(err)

	err = ValidateDistributeFeeFlags(
		cmd,
		[]string{},
	)
	f.Nil(err)
}

func (f *FeeHandlerTestSuite) TestValidateDistributeFeeInvalidFlagsFeeHandlerWithFeeOracle() {
	cmd := new(cobra.Command)
	BindDistributeFeeFlags(cmd)

	err := cmd.Flag("fee-handler").Value.Set(validAddr)
	f.Nil(err)
	err = cmd.Flag("distribution-array").Value.Set(fmt.Sprintf("%s,10", validAddr))
	f.Nil(err)
	err = cmd.Flag("fee-handler-with-oracle").Value.Set("true")
	f.Nil(err)

	err = ValidateDistributeFeeFlags(
		cmd,
		[]string{},
	)
	f.NotNil(err)
}
