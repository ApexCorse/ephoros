import "dart:convert";

import "package:client/types/record.dart";
import "package:http/http.dart" as http;

class API {
  const API({
    required http.Client httpClient,
    required String baseUrl,
    required String token,
  }) : _httpClient = httpClient,
       _baseUrl = baseUrl,
       _token = token;

  final http.Client _httpClient;
  final String _baseUrl;
  final String _token;

  Future<List<Record>> getRecordsBetweenTimeRange({
    required String section,
    required String module,
    required String sensor,
    DateTime? start,
    DateTime? end,
  }) async {
    final body = <String, String>{
      "section": section,
      "module": module,
      "sensor": sensor,
    };

    if (start != null) {
      body["start"] = start.toIso8601String();
    }
    if (end != null) {
      body["end"] = end.toIso8601String();
    }

    final response = await _httpClient.post(
      Uri.parse("$_baseUrl/data"),
      headers: {
        "Authorization": "Bearer $_token",
        "Content-Type": "application/json",
      },
      body: jsonEncode(body),
    );

    switch (response.statusCode) {
      case 401:
        throw const APIException("Unauthorized");
      case 400:
        throw const APIException("Bad Request");
    }

    return (jsonDecode(response.body) as List)
        .map((e) => Record.fromJson(e as Map<String, dynamic>))
        .toList();
  }
}

class APIException implements Exception {
  const APIException(this.message);

  final String message;
}
