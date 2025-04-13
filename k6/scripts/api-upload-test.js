import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const uploadedBytes = new Counter('s3_uploaded_bytes');
const successRate = new Rate('s3_success_rate');
const uploadDuration = new Trend('s3_upload_duration');
const timeToFirstByte = new Trend('s3_time_to_first_byte');
const apiLatency = new Trend('api_latency');

// Get test parameters from environment or use defaults
const apiBaseUrl = __ENV.API_BASE_URL || 'http://localhost:8080';
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

// Fetch a presigned URL from the API
function getPresignedUrl() {
    const apiStartTime = new Date().getTime();

    // Call the API to get a presigned URL
    const response = http.get(`${apiBaseUrl}/api/v1/presigned-url`, {
        tags: { name: 'GetPresignedURL' }
    });

    // Record API latency
    const apiEndTime = new Date().getTime();
    apiLatency.add(apiEndTime - apiStartTime);

    // Check if the request was successful
    const success = response.status === 200;
    if (!success) {
        if (__ENV.VERBOSE) {
            console.log(`Failed to get presigned URL: ${response.status} ${response.body}`);
        }
        return null;
    }

    // Parse the response
    try {
        return JSON.parse(response.body);
    } catch (e) {
        if (__ENV.VERBOSE) {
            console.log(`Failed to parse API response: ${e.message}`);
        }
        return null;
    }
}

// Main test function
export default function () {
    // Get a presigned URL from the API
    const urlData = getPresignedUrl();
    if (!urlData) {
        // If we couldn't get a URL, abort this iteration
        sleep(1);
        return;
    }

    // Extract URL and key
    const url = urlData.url;
    const key = urlData.key;

    if (!url) {
        console.log('Invalid URL from API');
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