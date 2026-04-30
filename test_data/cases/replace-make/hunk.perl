Config::Patch::Hunk->new(
   key  => "myapp",
   mode => "replace",
   regex => qr(^all:.*?\n\n)sm,
   text => "all:\n\techo 'all is gone!'\n",
);
