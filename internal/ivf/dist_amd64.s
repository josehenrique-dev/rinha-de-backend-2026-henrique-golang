#include "textflag.h"

#define STEP(qoff, boff) \
	VPBROADCASTW qoff(AX), X2 \
	VPMOVSXWD X2, Y2 \
	VMOVDQU boff(BX), X3 \
	VPMOVSXWD X3, Y3 \
	VPSUBD Y3, Y2, Y4 \
	VPMULLD Y4, Y4, Y4 \
	VPMOVSXDQ X4, Y5 \
	VEXTRACTI128 $1, Y4, X6 \
	VPMOVSXDQ X6, Y6 \
	VPADDQ Y5, Y0, Y0 \
	VPADDQ Y6, Y1, Y1

TEXT ·quantizedBlock8DistancesAVX2(SB),NOSPLIT,$0-24
	MOVQ query+0(FP), AX
	MOVQ block+8(FP), BX
	MOVQ out+16(FP), CX

	VPXOR Y0, Y0, Y0
	VPXOR Y1, Y1, Y1

	STEP(10, 80)
	STEP(12, 96)
	STEP(4, 32)
	STEP(0, 0)
	STEP(14, 112)
	STEP(16, 128)
	STEP(24, 192)
	STEP(2, 16)
	STEP(6, 48)
	STEP(8, 64)
	STEP(18, 144)
	STEP(20, 160)
	STEP(22, 176)
	STEP(26, 208)

	VMOVDQU Y0, 0(CX)
	VMOVDQU Y1, 32(CX)
	VZEROUPPER
	RET

#define BLOCK8(blockoff, outoff) \
	VPXOR Y0, Y0, Y0 \
	VPXOR Y1, Y1, Y1 \
	STEP(10, blockoff+80) \
	STEP(12, blockoff+96) \
	STEP(4,  blockoff+32) \
	STEP(0,  blockoff+0) \
	STEP(14, blockoff+112) \
	STEP(16, blockoff+128) \
	STEP(24, blockoff+192) \
	STEP(2,  blockoff+16) \
	STEP(6,  blockoff+48) \
	STEP(8,  blockoff+64) \
	STEP(18, blockoff+144) \
	STEP(20, blockoff+160) \
	STEP(22, blockoff+176) \
	STEP(26, blockoff+208) \
	VMOVDQU Y0, outoff(CX) \
	VMOVDQU Y1, outoff+32(CX)

TEXT ·quantizedBlock32DistancesAVX2(SB),NOSPLIT,$0-24
	MOVQ query+0(FP), AX
	MOVQ block+8(FP), BX
	MOVQ out+16(FP), CX

	BLOCK8(0,   0)
	BLOCK8(224, 64)
	BLOCK8(448, 128)
	BLOCK8(672, 192)

	VZEROUPPER
	RET

#undef BLOCK8
#undef STEP

#define ROWSTEP(qoff, boff) \
	VPBROADCASTW qoff(AX), X2 \
	VPMOVSXWD X2, Y2 \
	VPINSRW $0, boff+0(BX),   X3, X3 \
	VPINSRW $1, boff+28(BX),  X3, X3 \
	VPINSRW $2, boff+56(BX),  X3, X3 \
	VPINSRW $3, boff+84(BX),  X3, X3 \
	VPINSRW $4, boff+112(BX), X3, X3 \
	VPINSRW $5, boff+140(BX), X3, X3 \
	VPINSRW $6, boff+168(BX), X3, X3 \
	VPINSRW $7, boff+196(BX), X3, X3 \
	VPMOVSXWD X3, Y3 \
	VPSUBD Y3, Y2, Y4 \
	VPMULLD Y4, Y4, Y4 \
	VPMOVSXDQ X4, Y5 \
	VEXTRACTI128 $1, Y4, X6 \
	VPMOVSXDQ X6, Y6 \
	VPADDQ Y5, Y0, Y0 \
	VPADDQ Y6, Y1, Y1 \
	VPXOR X3, X3, X3

TEXT ·quantized8DistancesRowMajorAVX2(SB),NOSPLIT,$0-24
	MOVQ query+0(FP), AX
	MOVQ vectors+8(FP), BX
	MOVQ out+16(FP), CX

	VPXOR Y0, Y0, Y0
	VPXOR Y1, Y1, Y1
	VPXOR X3, X3, X3

	ROWSTEP(10, 10)
	ROWSTEP(12, 12)
	ROWSTEP(4, 4)
	ROWSTEP(0, 0)
	ROWSTEP(14, 14)
	ROWSTEP(16, 16)
	ROWSTEP(24, 24)
	ROWSTEP(2, 2)
	ROWSTEP(6, 6)
	ROWSTEP(8, 8)
	ROWSTEP(18, 18)
	ROWSTEP(20, 20)
	ROWSTEP(22, 22)
	ROWSTEP(26, 26)

	VMOVDQU Y0, 0(CX)
	VMOVDQU Y1, 32(CX)
	VZEROUPPER
	RET
