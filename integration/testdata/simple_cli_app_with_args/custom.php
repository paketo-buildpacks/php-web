<?php
  while (true) {
    fwrite(STDOUT, ($argv[1] ? $argv[1] : "SUCCESS") . PHP_EOL);
    sleep(10);
  }
?>
