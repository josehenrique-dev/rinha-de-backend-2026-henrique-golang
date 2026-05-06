#include "textflag.h"

// func squaredDist14(a, b []float32) float32
//
// Register calling convention (Go 1.17+ ABI):
//   a.ptr → AX   a.len → BX   a.cap → CX
//   b.ptr → DI   b.len → SI   b.cap → R8
//   return float32 → X0
//
// Layout of 14 float32s (56 bytes):
//   [0..7]  → Y0/Y1 (YMM, 32 bytes)
//   [8..11] → X3/X4 (XMM, 16 bytes, offset 32)
//   [12..13]→ X5/X6 (low 64 bits via VMOVSD, offset 48)
//
TEXT ·squaredDist14(SB), NOSPLIT, $0-56
    // Block [0..7]: load 8 floats into YMM registers
    VMOVUPS (AX), Y0          // a[0:8]
    VMOVUPS (DI), Y1          // b[0:8]
    VSUBPS  Y1, Y0, Y1        // Y1 = a[0:8] - b[0:8]
    VMULPS  Y1, Y1, Y2        // Y2 = (a[0:8]-b[0:8])^2

    // Block [8..11]: load 4 floats into XMM registers
    VMOVUPS 32(AX), X3        // a[8:12]
    VMOVUPS 32(DI), X4        // b[8:12]
    VSUBPS  X4, X3, X3        // X3 = a[8:12] - b[8:12]
    VMULPS  X3, X3, X3        // X3 = (a[8:12]-b[8:12])^2

    // Block [12..13]: VMOVSD loads 8 bytes, zeros upper 64 bits of XMM
    VMOVSD  48(AX), X5        // a[12], a[13] in low 64 bits; upper = 0
    VMOVSD  48(DI), X6        // b[12], b[13] in low 64 bits; upper = 0
    VSUBPS  X6, X5, X5        // X5 = a[12:14]-b[12:14], upper 2 floats = 0
    VMULPS  X5, X5, X5        // X5 = (a[12:14]-b[12:14])^2

    // Reduce Y2 (8 floats) → 4 floats in X7
    // X2 is the lower 128 bits of Y2 (same physical register)
    VEXTRACTF128 $1, Y2, X7   // X7 = Y2[4:8]
    VADDPS  X7, X2, X7        // X7 = Y2[0:4] + Y2[4:8]  (4 partial sums)

    // Accumulate [8..11] and [12..13]
    VADDPS  X3, X7, X7        // X7 += (a[8:12]-b[8:12])^2
    VADDPS  X5, X7, X7        // X7 += (a[12:14]-b[12:14])^2 (upper 2 = 0, safe)

    // Horizontal sum: 4 floats → 1 float
    VHADDPS X7, X7, X7        // X7[0]=X7[0]+X7[1], X7[1]=X7[2]+X7[3]
    VHADDPS X7, X7, X0        // X0[0] = X7[0]+X7[1] = total squared distance

    VZEROUPPER
    RET
