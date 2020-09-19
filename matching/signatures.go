package matching

import (
	"encoding/json"
	"fmt"
	"gitrob/common"
	"io/ioutil"
)

type Signatures struct {
	FileSignatures    []FileSignature
	ContentSignatures []ContentSignature
}

func (s *Signatures) loadSignatures(path string) error {
	if !common.FileExists(path) {
		return fmt.Errorf("missing signature file: %s", path)
	}
	data, readError := ioutil.ReadFile(path)
	if readError != nil {
		return readError
	}
	if unmarshalError := json.Unmarshal(data, &s); unmarshalError != nil {
		return unmarshalError
	}
	return nil
}

func (s *Signatures) Load(mode int) error {
	var e error
	if mode != ModeContentMatch {
		e = s.loadSignatures("./filesignatures.json")
		if e != nil {
			return e
		}
	}
	if mode != ModeFileMatch {
		//source:  https://github.com/dxa4481/truffleHogRegexes/blob/master/truffleHogRegexes/regexes.json
		e = s.loadSignatures("./contentsignatures.json")
		if e != nil {
			return e
		}
	}
	return nil
}
