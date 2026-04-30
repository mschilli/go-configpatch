Config::Patch::Hunk->new(
   key  => "myapp",
   mode => "replace",
   regex => qr(foo),
   text => "HELLO",
);
