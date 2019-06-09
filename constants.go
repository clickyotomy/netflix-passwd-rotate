package main

const (
	// netflixPasswordRoute is the default URL for logging into Netflix.
	netflixPasswordRoute = "https://netflix.com/password"

	// netflixMount is the base XPath for the page.
	netflixMnt = `//*[@id="appMountPoint"]`

	// netflixEval is a JavaScript expression to evaluate.
	netflixEval = `
	document.evaluate(
		'%s', document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null
	).singleNodeValue%s
	`

	// netflixVerifyWait is the maximum number of seconds to wait before
	// evaluating the verify expression.
	netflixVerifyWait = 4

	// autoGenerateSymsAll has all the special characters password generation.
	autoGenerateSymsAll = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

	// autoGenerateSymsTest has a limited set of special characters, because
	// sometimes it is cumbersome to pass certain characters as command line
	// arguments. This skips `~' (tilde) as well.
	// This is used for testing only.
	autoGenerateSymsTest = "#%&()*+,-./:;<>?@[\\]^_{|}"

	// Errors.
	errExecFail   = 1 // Browser task execution failed.
	errVerifyFail = 2 // Verification failed.
	errLoginFail  = 3 // Login failed.
	errUpdateFail = 4 // Update failed.
	errFlagFail   = 5 // CLI options parsing or user input failed.
	errAutoFail   = 6 // Password generation failed.
	errTmpFail    = 7 // Creation of temporary directory failed.
	errWriteFail  = 8 // File I/O failures.
)
