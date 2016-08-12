#!/bin/sh


go build -o fft *.go || exit

./fft -tone -raw -samples "$2" "$1"
exit
echo ========== Unpacking
./fft -raw -samples "$2" "$1.tsv"


sox -r 44100 -b 16 -c 1 -L -e signed-integer "$1.tsv.rev.raw" -b 16 -r 44100 "$1.rev.wav"
sox "$1.rev.wav" -n spectrogram -Y 130
open spectrogram.png

play -r 44100 -b 16 -c 1 -L -e signed-integer "$1.tsv.rev.raw"
