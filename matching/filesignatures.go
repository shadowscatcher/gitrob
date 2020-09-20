package matching

import (
	"fmt"
	"regexp"
)

const (
	ExtensionSignatureType = "extension"
	FilenameSignatureType  = "filename"
	PathSignatureType      = "path"
)

type FileSignature struct {
	Part        string
	MatchOn     string
	Description string
	Comment     string
}

func (f FileSignature) Match(target MatchTarget) (bool, error) {
	var haystack *string
	switch f.Part {
	case PathSignatureType:
		haystack = &target.Path
	case FilenameSignatureType:
		haystack = &target.Filename
	case ExtensionSignatureType:
		haystack = &target.Extension
	default:
		return false, fmt.Errorf("unrecognized 'Part' parameter: %s", f.Part)
	}
	return regexp.MatchString(f.MatchOn, *haystack)
}

func (f FileSignature) GetDescription() string {
	return f.Description
}

func (f FileSignature) GetComment() string {
	return f.Comment
}
