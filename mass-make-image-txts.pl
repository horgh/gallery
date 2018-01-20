# Create images.txt with empty descriptions for each subdirectory.
use strict;
use warnings;

my $dh;
opendir $dh, '.' or die $!;

while (my $filename = readdir $dh) {
  next if $filename eq '..' || $filename eq '.';
  next if $filename =~ /something-i-dont-want/;
  next unless -d $filename;
  print "$filename\n";
  chdir $filename;

  my $dh2;
  opendir $dh2, '.' or die $!;
  my @filenames;
  while (my $filename2 = readdir $dh2) {
    next if $filename2 eq '..' || $filename2 eq '.' || $filename2 =~ /\.MOV$/ ||
			$filename2 eq 'images.txt';
    push @filenames, $filename2;
  }
  closedir $dh2;

  chdir '..';

  my $fh;
  open $fh, '>', "$filename/images.txt" or die $!;

  my @sorted_filenames = sort { $a cmp $b } @filenames;
  foreach my $filename2 (@sorted_filenames) {
    print { $fh } "$filename2\n\n";
  }

  close $fh or die $!;

  print "wrote $filename/images.txt\n";
}

closedir $dh;
