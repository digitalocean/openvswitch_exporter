#!/bin/bash

# Verify that the correct license block is present in all Go source
# files.
EXPECTED=$(cat ./.scripts/license.txt)

# Scan each Go source file for license, excluding ./vendor directory.
EXIT=0
GOFILES=$(find . -path ./vendor -prune -o -name "*.go" -print)

for FILE in $GOFILES; do
	BLOCK=$(head -n 2 $FILE)

	echo $BLOCK

	if [ "$BLOCK" != "$EXPECTED" ]; then
		echo "file missing license: $FILE"
		EXIT=1
	fi
done

exit $EXIT
