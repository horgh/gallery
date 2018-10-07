#!/usr/bin/env perl
#
# Create images.txt for the given directory.

use strict;
use warnings;

sub main {
	if (@ARGV != 1 && @ARGV != 2) {
		print { \*STDERR } "Usage: $0 <directory> [initial description]\n" or die $!;
		print { \*STDERR } "\n" or die $!;
		print { \*STDERR } "  Initial description gets given to every image.\n";
		print { \*STDERR } "  Blank is okay.\n";
		print { \*STDERR } "\n" or die $!;
		return 0;
	}

	my $dir = $ARGV[0];
	my $description = $ARGV[1];

	my $dh;
	opendir $dh, $dir or die $!;

	my @filenames;
	while (my $filename = readdir $dh) {
		next if $filename eq '..' || $filename eq '.' ||
			$filename =~ /\.MOV$/i ||
			$filename eq 'images.txt' ||
			$filename =~ /\.mp4$/i ||
			$filename =~ /\.heic$/i;
		push @filenames, $filename;
	}

	closedir $dh;

	my @sorted_filenames = sort { $a cmp $b } @filenames;

	my $fh;
	open $fh, '>', "$dir/images.txt" or die $!;

	foreach my $filename (@sorted_filenames) {
		if (!$description) {
			print { $fh } "$filename\n\n" or die $!;
		} else {
			print { $fh } "$filename\n$description\n\n" or die $!;
		}
	}

	close $fh or die $!;

	print "wrote $dir/images.txt\n" or die $!;

	return 1;
}

exit(main() ? 0 : 1);
