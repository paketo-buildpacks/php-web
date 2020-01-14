Redis Loaded: <?php print(in_array("redis", get_loaded_extensions(), true)); ?><br/>
Memcached Loaded: <?php print(in_array("memcached", get_loaded_extensions(), true)); ?><br/>
Session Handler: <?php print(ini_get("session.save_handler")); ?><br/>
Session Name: <?php print(ini_get("session.name")); ?><br/>
Session Save Path: <?php print(ini_get("session.save_path")); ?><br/>
Memcached Session Binary: <?php print(ini_get("memcached.sess_binary_protocol")); ?><br/>
Memcached SASL User: <?php print(ini_get("memcached.sess_sasl_username")); ?><br/>
Memcached SASL Pass: <?php print(ini_get("memcached.sess_sasl_password")); ?><br/>
