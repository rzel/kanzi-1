/*
Copyright 2011-2017 Frederic Langlet
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
you may obtain a copy of the License at

                http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

#include <cstring>
#include "X86Codec.hpp"

using namespace kanzi;

bool X86Codec::forward(SliceArray<byte>& input, SliceArray<byte>& output, int count)
{
    if ((!SliceArray<byte>::isValid(input)) || (!SliceArray<byte>::isValid(output)))
        return false;

    if (input._array == output._array)
        return false;

    if ((count < 0) || (count + input._index > input._length))
        return false;

    // Aliasing
    byte* src = &input._array[input._index];
    byte* dst = &output._array[output._index];
    const int end = count - 8;
    int jumps = 0;

    for (int i = 0; i < end; i++) {
        if ((src[i] & INSTRUCTION_MASK) == INSTRUCTION_JUMP) {
           // Count valid relative jumps (E8/E9 .. .. .. 00/FF)
           if ((src[i+4] == 0) || (src[i+4] == -1)) {
              // No encoding conflict ?
              if ((src[i] != 0) && (src[i] != 1) && (src[i] != ESCAPE))
                 jumps++;
           }
        }
    }

    if (jumps < (count >> 7)) {
        // Number of jump instructions too small => either not a binary
        // or not worth the change => skip. Very crude filter obviously.
        // Also, binaries usually have a lot of 0x88..0x8C (MOV) instructions.
        return false;
    }

    int srcIdx = 0;
    int dstIdx = 0;

    while (srcIdx < end) {
        dst[dstIdx++] = src[srcIdx++];

        // Relative jump ?
        if ((src[srcIdx - 1] & INSTRUCTION_MASK) != INSTRUCTION_JUMP)
            continue;

        const byte cur = src[srcIdx];

        if ((cur == 0) || (cur == 1) || (cur == ESCAPE)) {
            // Conflict prevents encoding the address. Emit escape symbol
            dst[dstIdx] = ESCAPE;
            dst[dstIdx + 1] = cur;
            srcIdx++;
            dstIdx += 2;
            continue;
        }

        const byte sgn = src[srcIdx + 3];

        // Invalid sign of jump address difference => false positive ?
        if ((sgn != 0) && (sgn != -1))
            continue;

        int addr = (0xFF & src[srcIdx]) | ((0xFF & src[srcIdx + 1]) << 8) | 
                  ((0xFF & src[srcIdx + 2]) << 16) | ((0xFF & sgn) << 24);

        addr += srcIdx;
        dst[dstIdx] = byte(sgn + 1);
        dst[dstIdx + 1] = byte(ADDRESS_MASK ^ (0xFF & (addr >> 16)));
        dst[dstIdx + 2] = byte(ADDRESS_MASK ^ (0xFF & (addr >> 8)));
        dst[dstIdx + 3] = byte(ADDRESS_MASK ^ (0xFF & addr));
        srcIdx += 4;
        dstIdx += 4;
    }

    while (srcIdx < count)
        dst[dstIdx++] = src[srcIdx++];

    input._index = srcIdx;
    output._index = dstIdx;
    return true;
}

bool X86Codec::inverse(SliceArray<byte>& input, SliceArray<byte>& output, int count)
{
    if ((input._array == NULL) || (output._array == NULL) || (input._array == output._array))
        return false;

    if ((count < 0) || (count + input._index > input._length))
        return false;

    // Aliasing
    byte* src = &input._array[input._index];
    byte* dst = &output._array[output._index];
    const int end = count - 8;
    int srcIdx = 0;
    int dstIdx = 0;

    while (srcIdx < end) {
        dst[dstIdx++] = src[srcIdx++];

        // Relative jump ?
        if ((src[srcIdx - 1] & INSTRUCTION_MASK) != INSTRUCTION_JUMP)
            continue;

        const byte sgn = src[srcIdx];

        if (sgn == ESCAPE) {
            // Not an encoded address. Skip escape symbol
            srcIdx++;
            continue;
        }

        // Invalid sign of jump address difference => false positive ?
        if ((sgn != 1) && (sgn != 0))
            continue;

        int addr =  (0xFF & (ADDRESS_MASK ^ src[srcIdx+3]))        | 
                   ((0xFF & (ADDRESS_MASK ^ src[srcIdx+2])) <<  8) |
                   ((0xFF & (ADDRESS_MASK ^ src[srcIdx+1])) << 16) | 
                   ((0xFF & (sgn-1)) << 24);
        addr -= dstIdx;
        dst[dstIdx] = byte(addr);
        dst[dstIdx + 1] = byte(addr >> 8);
        dst[dstIdx + 2] = byte(addr >> 16);
        dst[dstIdx + 3] = byte(sgn - 1);
        srcIdx += 4;
        dstIdx += 4;
    }

    while (srcIdx < count)
        dst[dstIdx++] = src[srcIdx++];

    input._index = srcIdx;
    output._index = dstIdx;
    return true;
}
