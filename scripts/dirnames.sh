#!/bin/sh

# Print dirname of each relative pathname from stdin (net per line).

# pathname	dirname
# ----------------------
# a/b/c.go	a/b
# c.go		.
exec sed \
	-e 's:^:./:' \
	-e 's:/[^/]*$::' \
	-e 's:^\./::'
