#include "textflag.h"

TEXT ·distInt16(SB),NOSPLIT,$0-60
    MOVQ    v_base+32(FP), SI

    MOVOU   q+0(FP), X0
    MOVOU   0(SI), X1
    PSUBW   X1, X0
    PMADDWL X0, X0

    MOVOU   q+16(FP), X2
    MOVOU   16(SI), X3

    PCMPEQW X4, X4
    PSRLDQ  $4, X4
    PAND    X4, X2
    PAND    X4, X3

    PSUBW   X3, X2
    PMADDWL X2, X2

    PADDL   X2, X0

    PSHUFD  $0x4E, X0, X1
    PADDL   X1, X0
    PSHUFD  $0xB1, X0, X1
    PADDL   X1, X0

    MOVL    X0, ret+56(FP)
    RET
