Config::Patch::Hunk->new(
   key  => "myapp",
   mode => "insert-after",
   regex => qr(insert)m,
   text => "HELLO",
);
