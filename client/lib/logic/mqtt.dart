import "dart:async";

import "package:mqtt5_client/mqtt5_client.dart";

class MQTT {
  MQTT({required MqttClient client}) : _client = client, _streams = {};

  final MqttClient _client;

  final Map<String, StreamController<MqttMessage>> _streams;

  Future<void> connect({
    required String username,
    required String password,
    Duration timeout = const Duration(milliseconds: 5000),
  }) async {
    final connectionMessage = MqttConnectMessage()
        .authenticateAs(username, password)
        .startClean();
    _client.connectionMessage = connectionMessage;

    final previousOnConnected = _client.onConnected;
    final previousOnDisconnected = _client.onDisconnected;
    final previousOnFailed = _client.onFailedConnectionAttempt;

    final completer = Completer<void>();
    Timer? timeoutTimer;

    void cleanUp() {
      timeoutTimer?.cancel();
      _client.onConnected = previousOnConnected;
      _client.onDisconnected = previousOnDisconnected;
      _client.onFailedConnectionAttempt = previousOnFailed;
    }

    _client.onConnected = () {
      if (!completer.isCompleted) {
        completer.complete();
      }
    };

    _client.onFailedConnectionAttempt = (int attempt) {
      if (!completer.isCompleted) {
        completer.completeError(
          Exception("Failed to connect to MQTT broker: attempt $attempt"),
        );
      }
    };

    _client.onDisconnected = () {
      if (!completer.isCompleted) {
        completer.completeError(
          Exception("Disconnected before connection established"),
        );
      }
    };

    try {
      final connectFuture = _client.connect();

      timeoutTimer = Timer(timeout, () {
        if (!completer.isCompleted) {
          completer.completeError(
            Exception("MQTT connection timed out after $timeout"),
          );
        }
      });

      await connectFuture;
      await completer.future;
    } catch (e) {
      try {
        _client.disconnect();
      } catch (_) {}
      cleanUp();
      rethrow;
    }

    cleanUp();

    if (_client.connectionStatus?.state != MqttConnectionState.connected) {
      _client.disconnect();
      throw Exception("MQTT client not connected after connect procedure");
    }

    _startListening();
  }

  Stream<MqttMessage> subscribe(String topic) {
    if (_streams.containsKey(topic)) {
      return _streams[topic]!.stream;
    }

    _client.subscribe(topic, MqttQos.atMostOnce);

    final streamController = StreamController<MqttMessage>.broadcast();
    _streams[topic] = streamController;
    return streamController.stream;
  }

  void _startListening() {
    _client.updates.listen((updates) {
      for (final update in updates) {
        if (!_streams.containsKey(update.topic)) continue;
        if (update.payload is! MqttPublishMessage) continue;

        final message = update.payload as MqttPublishMessage;
        _streams[update.topic]?.add(message);
      }
    });
  }
}
