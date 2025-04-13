import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { SharedArray } from 'k6/data';

// Custom metrics
const uploadedBytes = new Counter('s3_uploaded_bytes');
const successRate = new Rate('s3_success_rate');
const uploadDuration = new Trend('s3_upload_duration');
const timeToFirstByte = new Trend('s3_time_to_first_byte');

// Load test data from file passed via --env flag
const testData = JSON.parse(open(__ENV.TEST_DATA_FILE));

// URLs array is shared across all VUs (virtual users)
const urls = new SharedArray('urls', function () {
    return testData.urls || [];
});

// Get test parameters from environment or use defaults
const concurrentUsers = __ENV.CONCURRENT_USERS || 10;
const testDurationSeconds = __ENV.TEST_DURATION_SECONDS || 60;
const rampUpDurationSeconds = __ENV.RAMP_UP_DURATION_SECONDS || 10;
const fileSize = __ENV.FILE_SIZE || 1048576; // 1MB
const contentType = __ENV.CONTENT_TYPE || 'application/octet-stream';

// Export K6 test configuration
export const options = {
    scenarios: {
        s3_uploads: {
            executor: 'ramping-vus',
            startVUs: 1,
            stages: [
                { duration: `${rampUpDurationSeconds}s`, target: concurrentUsers },
                { duration: `${testDurationSeconds}s`, target: concurrentUsers },
                { duration: '5s', target: 0 },
            ],
            gracefulRampDown: '5s',
        },
    },
    thresholds: {
        's3_success_rate': ['rate>0.95'], // 95% of uploads should succeed
        'http_req_duration': ['p(95)<5000'], // 95% of requests should complete within 5s
        'http_req_failed': ['rate<0.05'], // Less than 5% of requests should fail
    },
};

// Create a random payload of the specified size
function createRandomPayload(size) {
    const buffer = new Uint8Array(size);
    for (let i = 0; i < size; i++) {
        buffer[i] = Math.floor(Math.random() * 256);
    }
    return buffer;
}

// Main test function
export default function () {
    // Get a URL from the pool
    if (urls.length === 0) {
        console.log('No URLs available for testing');
        return;
    }

    // Pick a random URL from the pool
    const urlIndex = randomIntBetween(0, urls.length - 1);
    const urlEntry = urls[urlIndex];

    // Extract URL and key from the entry
    const url = urlEntry.url;
    const key = urlEntry.key;

    if (!url) {
        console.log(`Invalid URL entry at index ${urlIndex}`);
        return;
    }

    // Create random data payload
    const payload = createRandomPayload(fileSize);

    // Start the HTTP PUT request
    const params = {
        headers: {
            'Content-Type': contentType,
        },
        tags: {
            name: 'S3PutObject',
        },
    };

    // Track the start time for custom metrics
    const startTime = new Date().getTime();

    // Perform the upload
    const response = http.put(url, payload, params);

    // Calculate duration
    const duration = new Date().getTime() - startTime;

    // Check if the request was successful
    const success = response.status >= 200 && response.status < 300;

    // Record metrics
    successRate.add(success);
    uploadDuration.add(duration);
    timeToFirstByte.add(response.timings.firstByte);

    if (success) {
        uploadedBytes.add(fileSize);
    }

    // Verify the response
    check(response, {
        'status is 200': (r) => r.status === 200,
        'upload completed successfully': (r) => success,
    });

    // Log details in verbose mode
    if (__ENV.VERBOSE) {
        console.log(`Upload ${success ? 'succeeded' : 'failed'}: ${key}, Status: ${response.status}, Duration: ${duration}ms`);
    }

    // Add randomized sleep time between requests to simulate more realistic traffic
    sleep(randomIntBetween(0.1, 1));
} 