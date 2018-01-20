# Create images.txt for the given directory.
use strict;
use warnings;

sub main {
	if (@ARGV != 1) {
		print { \*STDERR } "Usage: $0 <directory>\n" or die $!;
		return 0;
	}

	my $dir = $ARGV[0];

	my $dh;
	opendir $dh, $dir or die $!;

	my @filenames;
	while (my $filename = readdir $dh) {
		next if $filename eq '..' || $filename eq '.' || $filename =~ /\.MOV$/ ||
			$filename eq 'images.txt';
		push @filenames, $filename;
	}

	closedir $dh;

	my @sorted_filenames = sort { $a cmp $b } @filenames;

	my $fh;
	open $fh, '>', "$dir/images.txt" or die $!;

	foreach my $filename (@sorted_filenames) {
		print { $fh } "$filename\n\n" or die $!;
	}

	close $fh or die $!;

	print "wrote $dir/images.txt\n" or die $!;

	return 1;
}

exit(main() ? 0 : 1);

