:global TITLE;
:global PASSPHRASE;

:local charset "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

:if ([:len $TITLE] < 3) do={ :error "Title is required"; }

:local passFile ($TITLE . ".pass")

:if ([:len [/file find name=$passFile]] > 0) do={
    :local stored [/file get [/file find name=$passFile] contents]
    :if ([:len $stored] >= 8) do={ :set PASSPHRASE $stored }
}

:if ([:len $PASSPHRASE] < 8) do={
    :set PASSPHRASE [:rndstr from=$charset length=10]
    :if ([:len [/file find name=$passFile]] = 0) do={ /file add name=$passFile type=file }
    /file set [/file find name=$passFile] contents=$PASSPHRASE
}

:if ([:len [/certificate find name=("cl-" . $TITLE)]] = 0) do={
    /certificate add name=("cl-" . $TITLE) copy-from=CLIENT-TEMPLATE common-name=("cl-" . $TITLE)
    /certificate sign ("cl-" . $TITLE) ca=LMTCA name=("cl-" . $TITLE)
    /certificate set ("cl-" . $TITLE) trusted=yes
    :log info ("[generate-certificate] Certificate cl-" . $TITLE . " has been created");
    :put ("certificate 'cl-" . $TITLE . "' has been created.");
}

:put ("OpenVPN Private Key Password: " . $PASSPHRASE);

:if ([:len [/file find name=("config-" . $TITLE . ".ovpn")]] = 0) do={
    /certificate export-certificate ("cl-" . $TITLE) export-passphrase=$PASSPHRASE file-name=$TITLE
    :delay 0.5
    :put ("[generate-certificate] Certificate exported as " . $TITLE . ".crt");
    :put ("[generate-certificate] Certificate key exported as " . $TITLE . ".key");

    /file remove [/file find name~"^client.*\\.ovpn\$"]

    /interface ovpn-server server export-client-configuration \
        server=TcpOpenVpn \
        server-address=vm4.sangesariha.ir \
        ca-certificate=LMTCA.crt \
        client-certificate=($TITLE . ".crt") \
        client-cert-key=($TITLE . ".key")

    :foreach fileId in=[/file find name~"^client.*\\.ovpn\$"] do={
        /file set $fileId name=("config-" . $TITLE . ".ovpn")
    }

    :put ("[generate-certificate] OpenVpn config exported as config-" . $TITLE . ".ovpn");
}
