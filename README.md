lang
====

Concurrent lisp-type language lexer, parser, optimizer  written in go.

## How it works

The `lexer.Lex()` takes a string, this could be a file or any other text string and is run concurrently on a channel. It chugs along on the string emitting tokens as it goes. The channel can then be given to `parser.Parse()` (along with a `chan *parser.Tree`), which takes these tokens and builds a parse tree, the parse tree is returned on the second channel upon completion. The parse tree can be optimized by handing it to `optim.Eval()`, which will return a `*optim.Tree`. `optim.Tree` has a `String()` interface, so you can just print it. If you want to do something else, you can just go through what's left in the tree (unknown variables and the pending/unknown keys).

## Implementation

If you would like to see an implementation of this, check out [Basilisk](http://github.com/cptaffe/basilisk), which is an interpreter which just takes file or input text and runs the three functions above on it. Minus some subtle printing, that is about all it does.

This code is licensed under a 2-clause BSD-style license that can be found in the LICENSE file.
