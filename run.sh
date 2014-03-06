#!/bin/bash

# Useage:
# ./run.sh

cd ${PWD}

echo 'updating git'
rm main.db
git pull
git checkout main.db
go build main.go

echo 'Killing old main'
for a in `ps -a | grep 'main'`
do
	if [[ "$a" != *[^0-9]* ]]&&[[ "$a" != 0* ]]
		then `kill -9 $a`
	fi
	break;
done

echo 'Starting run main...'
./main &
disown %1
echo 'Done'