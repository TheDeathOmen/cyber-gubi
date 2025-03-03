document.addEventListener("DOMContentLoaded", (event) => {
    startVideo();
});

function startVideo() {
    // Set desired dimensions
    const desiredWidth = 200;
    const desiredHeight = 150;
    navigator.getUserMedia(
        { video: { width: desiredWidth, height: desiredHeight } },
        stream => {
            video.srcObject = stream;
            video.width = desiredWidth;   // Set video width
            video.height = desiredHeight;  // Set video height
            canvas.width = desiredWidth;
            canvas.height = desiredHeight;
        },
        err => console.error(err)
    );
}

window.addEventListener('descriptorsFetched', (event) => {
    const jsonData = event.detail.descriptors;
    
    // Parse the JSON string into an object
    const parsedData = JSON.parse(jsonData);
    
    // Initialize an array to hold all reference descriptors as arrays
    const referenceDescriptors = [];

    // Iterate over each object in the parsed data
    parsedData.forEach((descriptor) => {
        // Extract values from the current object and push them to the referenceDescriptors array
        referenceDescriptors.push(Object.values(descriptor));
    });

    console.log(referenceDescriptors); // This will log an array of arrays
    initializeFaceRecognition(referenceDescriptors);
});

async function initializeFaceRecognition(referenceDescriptors) {
    const video = document.getElementById("video");
    const canvas = document.getElementById("canvas");
    const registerButton = window.parent.document.getElementById('register-btn');
    const loginButton = window.parent.document.getElementById('login-btn');
    const confidenceThreshold = 0.90;
    const distanceThreshold = 0.6; // Adjust as needed
    const numDescriptorsToCollect = 100; // Number of descriptors to collect
    let collectedDescriptors = []; // Array to store descriptors
    let isCollecting = false; // Flag to indicate if collecting is in progress
    let intervalId; // Variable to store the interval ID
    let loginSuccessful = false; // ADDED: Flag to prevent multiple logins

    if (!(video instanceof HTMLVideoElement)) {
        console.error("Error: 'video' is NOT an HTMLVideoElement.", video);
        alert("Critical Error:  Video element is not valid. See console.");  // Make it obvious
        return; // Stop initialization
    }

    if (!video.srcObject) {
        console.error("Error: 'video' element has no srcObject (no stream).", video);
        alert("Critical Error: Video stream not connected. See console.");
        return;
    }

    if (!video || !canvas || !registerButton || !loginButton) {
        console.error('Video, canvas, or button element not found!');
        return;
    }

    const ctx = canvas.getContext('2d');

    if (!ctx) {
        console.error('Could not get 2D context!');
        return;
    }

    // Load face-api.js models
    Promise.all([
        faceapi.nets.tinyFaceDetector.loadFromUri('/web/models'),
        faceapi.nets.faceLandmark68Net.loadFromUri('/web/models'),
        faceapi.nets.faceRecognitionNet.loadFromUri('/web/models'),
        faceapi.nets.faceExpressionNet.loadFromUri('/web/models')
    ]);

    // Function to calculate Euclidean distance
    function calculateDistance(descriptor1, descriptor2) {
        if (!descriptor1 || !descriptor2) {
            console.warn("One or both descriptors are null/undefined in calculateDistance");
            return Infinity; // Or some other large value
        }
        return faceapi.euclideanDistance(descriptor1, descriptor2);
    }

    const videoWidth = video.videoWidth;
    const videoHeight = video.videoHeight;
    const displaySize = { width: videoWidth, height: videoHeight };
    faceapi.matchDimensions(canvas, displaySize);

    intervalId = setInterval(async () => {
        const detections = await faceapi.detectAllFaces(video, new faceapi.TinyFaceDetectorOptions()).withFaceLandmarks().withFaceDescriptors();
        const resizedDetections = faceapi.resizeResults(detections, displaySize);
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        // Check for multiple faces
        if (resizedDetections.length > 1) {
            console.warn("Multiple faces detected. Waiting for a single face.");
            ctx.font = "20px Arial";
            ctx.fillStyle = "red";
            ctx.fillText("Multiple faces detected. Please show only one face.", 10, 50);
            return;
        }

        if (resizedDetections.length === 0) {
            console.warn("No faces detected.");
            ctx.font = "20px Arial";
            ctx.fillStyle = "red";
            ctx.fillText("No faces detected.", 10, 50);
            return;
        }

        // Now we know resizedDetections.length === 1
        const detection = resizedDetections[0]; // Access the first (and only) element

        const box = detection.detection.box;
        const drawBox = new faceapi.draw.DrawBox(box, { label: 'Unknown' });
        drawBox.draw(canvas);

        if (detection.detection.score > confidenceThreshold) { // Check confidence
            if (referenceDescriptors.length > 0 && !loginSuccessful) {
                // USE A FOR...OF LOOP
                for (const refDescriptor of referenceDescriptors) {
                    const distance = calculateDistance(refDescriptor, detection.descriptor);

                    if (distance < distanceThreshold) {
                        // **CRITICAL: CLEAR INTERVAL FIRST**
                        clearInterval(intervalId);
                        console.log("Face recognition stopped after successful login.");

                        loginSuccessful = true; // ADDED: Set loginSuccessful

                        drawBox.options.label = 'Matched face';
                        drawBox.draw(canvas);

                        // Dispatch login event with the matched reference descriptor
                        const loginEvent = new CustomEvent('click', {
                            detail: {
                                descriptor: JSON.stringify(refDescriptor), // Send only the matched descriptor
                            }
                        });

                        loginButton.dispatchEvent(loginEvent);

                        console.log("Login event dispatched!", loginEvent.detail);

                        break; // Exit the for...of loop
                    }
                }
            }

            // New face detected
            drawBox.options.label = 'New Face - Processing...';
            drawBox.draw(canvas);

            if (!isCollecting && !loginSuccessful) {
                isCollecting = true;
                collectedDescriptors = []; // Reset the array
                console.log("Collecting new descriptors...");
            }

            if (isCollecting && collectedDescriptors.length < numDescriptorsToCollect) {
                collectedDescriptors.push(detection.descriptor);
                if (collectedDescriptors.length === numDescriptorsToCollect) {
                    // Collection complete
                    console.log("Descriptor collection complete.");

                    // Calculate the average descriptor
                    const averageDescriptor = calculateAverageDescriptor(collectedDescriptors);
                    if (averageDescriptor) {
                        // Dispatch custom event with the raw average descriptor
                        const registerEvent = new CustomEvent('click', {
                            detail: {
                                descriptor: JSON.stringify(Array.from(averageDescriptor)),
                            }
                        });

                        registerButton.dispatchEvent(registerEvent);

                        console.log("Registration data dispatched!", registerEvent.detail);

                        // Stop the interval
                        clearInterval(intervalId);
                        console.log("Face recognition stopped after successful registration.");
                    } else {
                        console.warn("Could not calculate average descriptor.  Registration aborted.");
                        isCollecting = false; // Reset collection
                    }
                }
            }
        } else {
            drawBox.options.label = `Confidence: ${detection.detection.score.toFixed(2)}`;
            drawBox.draw(canvas);
        }

    }, 100);

    // Function to calculate average descriptor
    function calculateAverageDescriptor(descriptors) {
        if (!descriptors || descriptors.length === 0) {
            return null;
        }

        const numDescriptors = descriptors.length;
        const descriptorSize = descriptors[0].length; // Should be 128

        const sumDescriptor = new Float32Array(descriptorSize);

        for (let i = 0; i < numDescriptors; i++) {
            const descriptor = descriptors[i];
            if (!descriptor || descriptor.length !== descriptorSize) {
                console.warn("Invalid descriptor length in calculateAverageDescriptor. Skipping.");
                continue; // Skip malformed descriptors
            }
            for (let j = 0; j < descriptorSize; j++) {
                sumDescriptor[j] += descriptor[j];
            }
        }

        // Calculate the average
        const averageDescriptor = new Float32Array(descriptorSize);
        for (let i = 0; i < descriptorSize; i++) {
            averageDescriptor[i] = sumDescriptor[i] / numDescriptors;
        }

        return averageDescriptor;
    }
}