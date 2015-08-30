use Test::More;
use t::Util;
use Capture::Tiny qw/ capture /;

t::Util::compile("go2xstest", <<EOF);
package main

import "fmt"

//go2xs function
func function() {
  fmt.Println("Hello World")
}
EOF

my ($stdout, $strerr) = capture {
    go2xstest::function();
};

is $stdout, "Hello World\n";

done_testing;
