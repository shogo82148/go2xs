use Test::More;
use t::Util;

t::Util::compile("go2xstest", <<EOF);
package main

//go2xs hello
func hello(str string) string {
  return "Hello " + str
}
EOF

is go2xstest::hello("World"), "Hello World";

done_testing;
