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

#ifndef _BlockDecompressor_
#define _BlockDecompressor_

#include <map>
#include <vector>
#include "../concurrent.hpp"
#include "../OutputStream.hpp"
#include "../Listener.hpp"
#include "../io/CompressedInputStream.hpp"

namespace kanzi 
{

   class BlockDecompressor 
   {
   public:
      static const int WARN_EMPTY_INPUT = -128;

      BlockDecompressor(map<string, string>& map);    

      ~BlockDecompressor();

      int call();

      bool addListener(Listener* bl);

      bool removeListener(Listener* bl);

      void dispose();


   private:
      static const int DEFAULT_BUFFER_SIZE = 32768;

      int _verbosity;
      bool _overwrite;
      string _inputName;
      string _outputName;
      string _codec;
      string _transform;
      int _blockSize;
      int _jobs;
      OutputStream* _os;
      CompressedInputStream* _cis;
      vector<Listener*> _listeners;

      static void printOut(const char* msg, bool print);

      static void notifyListeners(vector<Listener*>& listeners, const Event& evt);
   };

}
#endif