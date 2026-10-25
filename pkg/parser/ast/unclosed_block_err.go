package ast

type UnclosedBlockError struct {
	BlockName      string
	ExpectedCloser string
}

func (e *UnclosedBlockError) Error() string {
	return "unclosed " + e.BlockName + ": expected " + e.ExpectedCloser
}

func (e *UnclosedBlockError) Unwrap() error {
	return ErrUnclosedBlock
}

func NewUnclosedBlockError(blockName, expectedCloser string) error {
	return &UnclosedBlockError{
		BlockName:      blockName,
		ExpectedCloser: expectedCloser,
	}
}
