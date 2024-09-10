#!/bin/sh
# Writes 19 secrets  to the file secrets.json
COUNT=25
if [[ ! -z $1 ]]; then
  COUNT=$1
fi

echo "[" > secrets.json
for f in $(eval echo "{1..$COUNT}")
do
aa="$(uuidgen | sed 's/-//g')" 
echo   \"$aa\", >> secrets.json
done
aa="$(uuidgen | sed 's/-//g')"
echo   \"$aa\" >> secrets.json
echo "]" >> secrets.json
