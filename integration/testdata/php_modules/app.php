<?php


    foreach (get_loaded_extensions() as $ext) {
        print($ext . "\n");
    }

    while (true) {
        sleep(10);
    }
?>