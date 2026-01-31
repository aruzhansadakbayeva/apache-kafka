
set -e
BROKERS=("kafka-controller-0" "kafka-controller-1" "kafka-1" "kafka-2" "kafka-3")
BASE_DIR="$(pwd)/certs"

CA_DIR="$BASE_DIR/ca"
STORE_PASS="password"
KEY_PASS="password"
VALIDITY_DAYS=365

rm -rf "$BASE_DIR"
mkdir -p "$CA_DIR"

# CREATE CA

echo "==> Generating CA"
openssl req -new -x509 \
  -keyout "$CA_DIR/ca.key" \
  -out "$CA_DIR/ca.crt" \
  -days $VALIDITY_DAYS \
  -nodes \
  -subj "/CN=Kafka-CA"

# GENERATE CERTS

generate_cert () {
  NAME=$1
  DIR="$BASE_DIR/$NAME"
  mkdir -p "$DIR"

  echo "==> Generating keystore for $NAME"

  keytool -genkeypair \
    -alias "$NAME" \
    -keystore "$DIR/kafka.keystore.jks" \
    -storepass "$STORE_PASS" \
    -keypass "$KEY_PASS" \
    -keyalg RSA \
    -validity $VALIDITY_DAYS \
    -dname "CN=$NAME" \
    -ext SAN=DNS:$NAME

  echo "==> Creating CSR"
  keytool -certreq \
    -alias "$NAME" \
    -keystore "$DIR/kafka.keystore.jks" \
    -storepass "$STORE_PASS" \
    -file "$DIR/$NAME.csr"

  echo "==> Signing cert with CA"
  openssl x509 -req \
    -CA "$CA_DIR/ca.crt" \
    -CAkey "$CA_DIR/ca.key" \
    -in "$DIR/$NAME.csr" \
    -out "$DIR/$NAME.crt" \
    -days $VALIDITY_DAYS \
    -CAcreateserial

  echo "==> Importing CA into keystore"
  keytool -importcert \
    -alias CARoot \
    -keystore "$DIR/kafka.keystore.jks" \
    -storepass "$STORE_PASS" \
    -file "$CA_DIR/ca.crt" \
    -noprompt

  echo "==> Importing signed cert into keystore"
  keytool -importcert \
    -alias "$NAME" \
    -keystore "$DIR/kafka.keystore.jks" \
    -storepass "$STORE_PASS" \
    -file "$DIR/$NAME.crt" \
    -noprompt

  echo "==> Creating truststore"
  keytool -importcert \
    -alias CARoot \
    -keystore "$DIR/kafka.truststore.jks" \
    -storepass "$STORE_PASS" \
    -file "$CA_DIR/ca.crt" \
    -noprompt

  rm "$DIR/$NAME.csr"
}

for broker in "${BROKERS[@]}"; do
  generate_cert "$broker"
done

echo "======================================"
echo " Kafka SSL certificates generated"
echo " Base dir: $BASE_DIR"
echo "======================================"
