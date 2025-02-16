#!/bin/bash

npm run build

cd _build
python3 -m http.server