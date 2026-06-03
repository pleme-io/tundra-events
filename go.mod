module github.com/pleme-io/tundra-events

go 1.25

require github.com/pleme-io/errors-go v0.0.0

// TEMP local override — remove once errors-go is published (the proxy resolves it then).
replace github.com/pleme-io/errors-go => ../errors-go
