lang
====

Concurrent lisp-type language lexer, parser, optimizer  written in go.

I am calling it basilisk.

## How it works

The `lexer.Lex()` takes a string, this could be a file or any other text string and is run concurrently on a channel. It chugs along on the string emitting tokens as it goes. The channel can then be given to `parser.Parse()` (along with a `chan *parser.Tree`), which takes these tokens and builds a parse tree, the parse tree is returned on the second channel upon completion. The parse tree can be optimized by handing it to `optim.Eval()`, which will return a `*optim.Tree`. `optim.Tree` has a `String()` interface, so you can just print it. If you want to do something else, you can just go through what's left in the tree (unknown variables and the pending/unknown keys).

For more information, refer to the [wiki](../../wiki)

## Implementation

If you would like to see an implementation of this, check out [Basilisk](http://github.com/cptaffe/basilisk), which is an interpreter which just takes file or input text and runs the three functions above on it. Minus some subtle printing, that is about all it does.

## What is working

This section lists the things that work, some pieces of the library like `lang/token` will allow `lang/lexer` to lex more tokens than either `lang/parse` or `lang/optim` will actually allow to be evaluated.

### Operations and Constants

- `+` is add
- `-` is subtract
- `*` is multipy
- `/` is divide
- `assign` assigns a variable to a value (an unevaluated ast)
- `lambda` defines a function with a list of args the first argument, and the operations as the second
- `cmp` evaluates the first argument, if it is 1 it executes the second arg, if it isn't it executes the third
- `eq` and `lt` for equals and less than evaluate two numbers and return 0 or 1
- `time` returns the system time in nanoseconds
- `print` prints something
- `lazy` forces non-lazy evaluation on variables
- `eval` evaluates a string of basilisk as basilisk

### Recursion

This snippet would evaluate factorial 40 and then print it.

```lisp
(assign factorial 
  (lambda (list n) 
    (cmp n 1 
      (mul n (factorial (sub n 1))))))
(print "Factorial 40 is " (factorial 40))
```

## License

This code is licensed under a 2-clause BSD-style license that can be found in the LICENSE file.
