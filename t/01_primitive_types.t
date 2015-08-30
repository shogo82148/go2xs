use Test::More;
use t::Util;
use Parallel::ForkManager;
use Test::SharedFork;

# Run tests in other processes becase t::Util::compile cannot call twice in same process... :(
# ref. https://github.com/golang/go/issues/11100
my $pm = Parallel::ForkManager->new(1);

for my $type (qw/int8 uint8 int16 uint16 int32 uin32 int64 uint64 int uint float32 float64/) {
    my $pid = $pm->start and next;
    t::Util::compile("go2xs$type", <<"EOF");
package main

//go2xs add
func add(a $type, b $type) $type {
  return a + b
}
EOF

    is eval("go2xs${type}::add(2, 3)"), 5, "$type";
    $pm->finish;
}

for my $type (qw/int8 uint8 int16 uint16 int32 uin32 int64 uint64 int uint float32 float64/) {
    my $pid = $pm->start and next;
    t::Util::compile("go2xs$type", <<"EOF");
package main

//go2xs swap
func swap(a $type, b $type) ($type, $type) {
  return b, a
}
EOF

    is_deeply [eval("go2xs${type}::swap(2, 3)")], [3, 2], "$type";
    $pm->finish;
}

$pm->wait_all_children;

done_testing;
