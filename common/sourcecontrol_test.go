package common

import (
	"encoding/json"
	testingSupport "github.com/codeEmitter/gitrob/.testing_support"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"testing"
)

type SourceControlTestSuite struct {
	suite.Suite
	AllChanges []object.Change
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SourceControlTestSuite))
}

func (suite *SourceControlTestSuite) SetupTest() {
	fileContents := testingSupport.ReadFiles("../.testing_support/data/changes/")
	for _, content := range fileContents {
		c := object.Change{}
		err := json.Unmarshal(content, &c)
		if err != nil {
			suite.Fail("Unable to load json data:  %s", err)
		}
		suite.AllChanges = append(suite.AllChanges, c)
	}
}

func (suite *SourceControlTestSuite) TestChangeContentsAreConcatenatedAndReturned() {
	changeContent, err := GetChangeContent(&suite.AllChanges[2])
	suite.Equal(suite.T(), nil, err)
	suite.NotEqual(suite.T(), 0, len(changeContent))
}
