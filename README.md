# QOI transcoder

This is a QOI transcoder coded in pure Go
The qoiconv utility is also included. No qoibench though
Performance is quite slower than the C reference transcoder. Could get faster with some SIMD using cgoasm for the hashing

Performance numbers below. From qoibench for the C reference and go bench for this implementation. Test image is a 2kx2k image of r/place  
Encoding performance will drop if using an image that does not have an NRGBA color model
|    | Encode | Decode |
|----|--------|--------|
| C  | 19.6   | 14.7   |
| Go | 96.1   | 28.2   |