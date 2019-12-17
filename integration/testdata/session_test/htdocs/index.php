Redis Loaded: <?php print(in_array("redis", get_loaded_extensions(), true)); ?><br/>
Session Handler: <?php print(ini_get("session.save_handler")); ?><br/>
Session Name: <?php print(ini_get("session.name")); ?><br/>
Session Save Path: <?php print(ini_get("session.save_path")); ?><br/>
