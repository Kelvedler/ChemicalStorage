#!/bin/sh

./update.sh

if [ $? -ne 0 ]
then
  echo "Application update error"
  exit 1
else
  ./chemical_storage
fi

