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

#ifndef _BlockCompressor_
#define _BlockCompressor_

#include <map>
#include <vector>
#include "../concurrent.hpp"
#include "../InputStream.hpp"
#include "../Listener.hpp"
#include "../io/CompressedOutputStream.hpp"

namespace kanzi 
{

   class BlockCompressor
   {
   public:
      static const int WARN_EMPTY_INPUT = -128;

      BlockCompressor(map<string, string>& m);
      
      ~BlockCompressor();

      int call();

      bool addListener(Listener* bl);

      bool removeListener(Listener* bl);

      void dispose();

   private:
      static const int DEFAULT_BUFFER_SIZE = 32768;
      static const int DEFAULT_BLOCK_SIZE = 1024 * 1024;

      int _verbosity;
      bool _overwrite;
      bool _checksum;
      string _inputName;
      string _outputName;
      string _codec;
      string _transform;
      int _blockSize;
      int _level; // command line compression level
      int _jobs;
      InputStream* _is;
      CompressedOutputStream* _cos;
      vector<Listener*> _listeners;

      static void printOut(const char* msg, bool print);

      static void notifyListeners(vector<Listener*>& listeners, const Event& evt);

      static void getTransformAndCodec(int level, string tranformAndCodec[2]);
   };

}
#endif