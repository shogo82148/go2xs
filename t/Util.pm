package t::Util;
use File::Temp qw/tempdir/;
use Cwd::Guard qw/cwd_guard/;
use File::Basename;
use File::Spec;

our $xs2go = File::Spec->rel2abs(File::Spec->catfile(dirname(dirname(__FILE__)), qw/cli go2xs main.go/));

sub compile {
    my ($name, $gocode) = @_;
    my $dir = tempdir;#( CLEANUP => 1 );
    warn $dir;

    my $guard = cwd_guard($dir);

    open my $fh, '>', "test.go";
    print $fh $gocode;
    close $fh;

    system("go run $xs2go -name $name test.go") == 0 or die;
    system("perl Makefile.PL") == 0 or die;
    system("make") == 0 or die;

    eval "use blib '$dir'; use $name;";
}

1;
