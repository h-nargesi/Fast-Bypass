# generate-certificate
:global TITLE;
:global PASSPHRASE;

:local charset "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

:if ([:len $TITLE] < 3) do={ :error "Title is required"; }

:local certBase ("certificates/" . $TITLE)
:local passFile ($certBase . ".pass")
:local certFile ($certBase . ".crt")
:local keyFile ($certBase . ".key")
:local certName ("cl-" . $TITLE)
:local configFile ("open-vpns/config-" . $TITLE . ".ovpn")

:if ([:len [/file find name=$passFile]] > 0) do={
    :local stored [/file get [/file find name=$passFile] contents]
    :if ([:len $stored] >= 8) do={ :set PASSPHRASE $stored }
}

:if ([:len $PASSPHRASE] < 8) do={
    :set PASSPHRASE [:rndstr from=$charset length=10]
    :if ([:len [/file find name=$passFile]] = 0) do={ /file add name=$passFile type=file }
    /file set [/file find name=$passFile] contents=$PASSPHRASE
}

:if ([:len [/certificate find name=$certName]] = 0) do={
    /certificate add name=$certName copy-from=CLIENT-TEMPLATE common-name=$certName
    /certificate sign $certName ca=LMTCA name=$certName
    /certificate set $certName trusted=yes
    :log info ("[generate-certificate] Certificate " . $certName . " has been created");
    :put ("certificate '" . $certName . "' has been created.");
}

:put ("OpenVPN Private Key Password: " . $PASSPHRASE);

:if ([:len [/file find name=$configFile]] = 0) do={
    /certificate export-certificate $certName export-passphrase=$PASSPHRASE file-name=$certBase
    :delay 0.5
    :put ("[generate-certificate] Certificate exported as " . $TITLE . ".crt");
    :put ("[generate-certificate] Certificate key exported as " . $TITLE . ".key");

    /file remove [/file find name~"^client.*\\.ovpn\$"]

    /interface ovpn-server server export-client-configuration \
        server=TcpOpenVpn \
        server-address=vm4.photon-ai.ir \
        ca-certificate=certificates/LMTCA.crt \
        client-certificate=$certFile \
        client-cert-key=$keyFile

    :foreach fileId in=[/file find name~"^client.*\\.ovpn\$"] do={
        /file set $fileId name=$configFile
    }

    :put ("[generate-certificate] OpenVpn config exported as config-" . $TITLE . ".ovpn");
}
