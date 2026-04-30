Config::Patch::Hunk->new(
   key  => "myapp",
   mode => "insert-after",
   regex => qr/bar/,
   text => "HELLO",
);
