Config::Patch::Hunk->new(
   key  => "myapp",
   mode => "insert-before",
   regex => qr(^bar$)m,
   text => "HELLO",
);
