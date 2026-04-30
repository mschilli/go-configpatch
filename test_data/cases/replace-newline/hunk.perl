Config::Patch::Hunk->new(
   key  => "myapp",
   mode => "replace",
   regex => qr(bar\n),
   text => "HELLO",
);
