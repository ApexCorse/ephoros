import "dart:convert";

import "package:client/logic/api.dart";
import "package:flutter_test/flutter_test.dart";
import "package:http/http.dart";
import "package:http/testing.dart";

void main() {
  group("API", () {
    group("getRecordsBetweenTimeRange", () {
      test("should return a list of records", () async {
        final api = API(
          httpClient: MockClient((request) async {
            expect(request.url.toString(), "https://api.example.com/data");
            expect(request.headers["Authorization"], "Bearer valid-token");
            expect(
              request.body,
              jsonEncode({
                "section": "section",
                "module": "module",
                "sensor": "sensor",
              }),
            );

            return Response(
              jsonEncode([
                {
                  "id": 1,
                  "section": "section",
                  "module": "module",
                  "sensor": "sensor",
                  "date": "2021-01-01T00:00:00Z",
                  "value": 1,
                },
                {
                  "id": 1,
                  "section": "section",
                  "module": "module",
                  "sensor": "sensor",
                  "date": "2021-01-01T00:00:01Z",
                  "value": 2,
                },
                {
                  "id": 1,
                  "section": "section",
                  "module": "module",
                  "sensor": "sensor",
                  "date": "2021-01-01T00:00:02Z",
                  "value": 3,
                },
              ]),
              200,
            );
          }),
          baseUrl: "https://api.example.com",
          token: "valid-token",
        );

        final records = await api.getRecordsBetweenTimeRange(
          section: "section",
          module: "module",
          sensor: "sensor",
        );

        expect(records.length, 3);
        expect(records.every((e) => e.section == "section"), isTrue);
        expect(records.every((e) => e.module == "module"), isTrue);
        expect(records.every((e) => e.sensor == "sensor"), isTrue);

        expect(records[0].date, DateTime.parse("2021-01-01T00:00:00Z"));
        expect(records[0].value, 1);
        expect(records[1].date, DateTime.parse("2021-01-01T00:00:01Z"));
        expect(records[1].value, 2);
        expect(records[2].date, DateTime.parse("2021-01-01T00:00:02Z"));
        expect(records[2].value, 3);
      });

      test("throws APIException on 401 Unauthorized", () async {
        final api = API(
          httpClient: MockClient(
            (request) async => Response("Unauthorized", 401),
          ),
          baseUrl: "https://api.example.com",
          token: "invalid-token",
        );

        expect(
          () => api.getRecordsBetweenTimeRange(
            section: "section",
            module: "module",
            sensor: "sensor",
          ),
          throwsA(
            isA<APIException>().having(
              (e) => e.message,
              "message",
              "Unauthorized",
            ),
          ),
        );
      });

      test("throws APIException on 400 Bad Request", () async {
        final api = API(
          httpClient: MockClient(
            (request) async => Response("Bad Request", 400),
          ),
          baseUrl: "https://api.example.com",
          token: "valid-token",
        );

        expect(
          () => api.getRecordsBetweenTimeRange(
            section: "section",
            module: "module",
            sensor: "sensor",
          ),
          throwsA(
            isA<APIException>().having(
              (e) => e.message,
              "message",
              "Bad Request",
            ),
          ),
        );
      });
    });
  });
}
