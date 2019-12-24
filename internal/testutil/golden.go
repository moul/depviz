package testutil

import "flag"

var update = flag.Bool("update", false, "update golden files")

func UpdateGolden() bool {
	return *update
}
