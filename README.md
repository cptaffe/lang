lang
====

Concurrent lisp-type language lexer, parser, optimizer  written in go.

## How it works

The `lexer.Lex()` takes a string, this could be a file or any other text string and is run concurrently on a channel. It chugs along on the string emitting tokens as it goes. The channel can then be given to `parser.Parse()` (along with a `chan *parser.Tree`), which takes these tokens and builds a parse tree, the parse tree is returned on the second channel upon completion. The parse tree can be optimized by handing it to `optim.Eval()`, which will return a `*optim.Tree`. `optim.Tree` has a `String()` interface, so you can just print it. If you want to do something else, you can just go through what's left in the tree (unknown variables and the pending/unknown keys).

For more information, refer to the [wiki](/wiki)

## Implementation

If you would like to see an implementation of this, check out [Basilisk](http://github.com/cptaffe/basilisk), which is an interpreter which just takes file or input text and runs the three functions above on it. Minus some subtle printing, that is about all it does.

## What is working

This section lists the things that work, some pieces of the library like `lang/token` will allow `lang/lexer` to lex more tokens than either `lang/parse` or `lang/optim` will actually allow to be evaluated.

### Operations and Constants

The only functioning operators as of now are the `add`, `sub`, `mul`, and `div` operators. The only numbers that currently will be evaluated are the ones `strconv.ParseFloat` in `lang/optim` will allow. This means that `1` will work as well as `0.3`, but while `0x4` and `'a'` will lex & parse, optim will log an error and return.

### Recursion

Other langauge functions that work are `assign` and `lambda` as well as `cmp` and `eq`, this allow things like a recursive factorial function such as this:

```lisp
(assign factorial 
  (lambda (list n) 
    (cmp n 1 
      (mul n (factorial (sub n 1))))))
```

__Unfortunately__ this only works for `(factorial 1)` and `(factorial 2)` for `(factorial 3)` it dies for some reason when being evaluated. Once it gets to `(*optim.eval).lambda` while figuring out the answer for `(factorial 2)`, and goes all the way to a call to `(*optim.eval).evaluate` to evaluate the contents of factorial when n = 2, and somehow seems to return without even calling the function. Some magical pending-call max wall kills it off or something, I have no idea what happends but I have traced it to that call.

## License

This code is licensed under a 2-clause BSD-style license that can be found in the LICENSE file.
