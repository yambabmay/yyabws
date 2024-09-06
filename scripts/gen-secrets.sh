#!/bin/sh
echo "[" > secrets.json
for f in {1..19}
do
aa="$(uuidgen | sed 's/-//g')" 
echo   \"$aa\", >> secrets.json
done
echo   \"$aa\" >> secrets.json
echo "]" >> secrets.json
