package flagvar

import (
	"os"
)

// File is a flag.Value for file paths. Returns any errors from os.Stat.
type File struct {
	Value string
}

// Help returns a string to include in the flag's help message.
func (f *File) Help() string {
	return "file path"
}

// Set implements flag.Value by checking for the file's existence through
// using os.Stat. Any error returned by os.Stat is returned by this function.
func (f *File) Set(v string) error {
	_, err := os.Stat(v)
	f.Value = v

	return err
}

// String implements flag.Value by returning the current file path.
func (f *File) String() string {
	if f == nil {
		return ""
	}

	return f.Value
}

// Type implements pflag.Value by noting our Value is string typed.
func (f *File) Type() string {
	return "string"
}
