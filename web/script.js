document.addEventListener("DOMContentLoaded", (event) => {
    startVideo();
});

function startVideo() {
    // Set desired dimensions
    const desiredWidth = 225;
    const desiredHeight = 225;
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
    
    // Initialize an object to hold all reference descriptors
    const referenceDescriptors = {};

    // Iterate over each object in the parsed data
    Object.keys(parsedData).forEach(key => {
        // console.log(`Key: ${key}, Values: ${parsedData[key]}`);
        // Extract values from the current object and store them in referenceDescriptors
        referenceDescriptors[key] = parsedData[key];
    });

    console.log(referenceDescriptors); // This will log an array of arrays
    initializeFaceRecognition(referenceDescriptors);
});

async function initializeFaceRecognition(referenceDescriptors) {
    const video = document.getElementById("video");
    const canvas = document.getElementById("canvas");
    const registerButton = window.parent.document.getElementById('register-btn');
    const loginButton = window.parent.document.getElementById('login-btn');
    const confidenceThreshold = 0.70;
    const distanceThreshold = 0.6; // Adjust as needed
    const numDescriptorsToCollect = 10; // Number of descriptors to collect
    let collectedDescriptors = []; // Array to store descriptors
    let isCollecting = false; // Flag to indicate if collecting is in progress
    let intervalId; // Variable to store the interval ID
    let loginSuccessful = false; // ADDED: Flag to prevent multiple logins

    // Liveness challenge variables
    let challenge = null;
    let challengeStartTime = 0;
    const CHALLENGE_TIMEOUT = 30000; // Time to complete the challenge
    let challengeSuccess = true;  // Track challenge success, change to false once done with testing
    // Constants for head nod
    const HEAD_NOD_HISTORY_LENGTH = 5;  // Number of frames to track head position
    const NOD_THRESHOLD = 3;  // Adjust this value based on testing
    let headPositionHistory = []; // Array to store head positions
    let nodDetected = false;    // Flag to indicate if nod is detected

    // Function to generate a random challenge
    function generateChallenge() {
        const challenges = ["nod"];
        const randomIndex = Math.floor(Math.random() * challenges.length);
        return challenges[randomIndex];
    }

    if (!(video instanceof HTMLVideoElement)) {
        console.error("Error: 'video' is NOT an HTMLVideoElement.", video);
        // alert("Critical Error:  Video element is not valid. See console.");  // Make it obvious
        location.reload();
        return; // Stop initialization
    }

    if (!video.srcObject) {
        console.error("Error: 'video' element has no srcObject (no stream).", video);
        // alert("Critical Error: Video stream not connected. See console.");
        location.reload();
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
    await Promise.all([
        faceapi.nets.tinyFaceDetector.loadFromUri('/web/models'),
        faceapi.nets.faceLandmark68Net.loadFromUri('/web/models'),
        faceapi.nets.faceRecognitionNet.loadFromUri('/web/models'),
        faceapi.nets.faceExpressionNet.loadFromUri('/web/models')
    ]);

    // Instead of global 'tf', use:
    const tf = faceapi.tf;
    // Load anti-spoofing model
    const spoofModel = await tf.loadGraphModel('/web/models/anti-spoofing.json');

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
        const detections = await faceapi.detectAllFaces(video, new faceapi.TinyFaceDetectorOptions()).withFaceLandmarks().withFaceDescriptors().withFaceExpressions(); // Added face expressions
        const resizedDetections = faceapi.resizeResults(detections, displaySize);
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        // Check for multiple faces
        if (resizedDetections.length > 1) {
            console.warn("Multiple faces detected. Waiting for a single face.");
            ctx.font = "20px Arial";
            ctx.fillStyle = "red";
            // ctx.fillText("Multiple faces detected. Please show only one face.", 10, 50);
            return;
        }

        if (resizedDetections.length === 0) {
            console.warn("No faces detected.");
            ctx.font = "20px Arial";
            ctx.fillStyle = "red";
            // ctx.fillText("No faces detected.", 10, 50);
            return;
        }

        // Now we know resizedDetections.length === 1
        const detection = resizedDetections[0]; // Access the first (and only) element
        const box = detection.detection.box;
        const drawBox = new faceapi.draw.DrawBox(box, { label: 'Unknown' });
        drawBox.draw(canvas);

        const SPOOF_INPUT_SIZE = 128; // 128x128 pixels
        const SPOOF_THRESHOLD = 0.8; // 80% confidence threshold

        // Anti-spoofing processing
        let isRealFace = true; // change to flase once done with testing
        // try {
        //     // Extract face region
        //     const regions = await faceapi.extractFaces(video, [detection.detection.box]);
        //     if (regions && regions.length > 0) {
        //         // Convert to tensor and preprocess
        //         const tensor = faceapi.tf.browser.fromPixels(regions[0])
        //             .resizeBilinear([SPOOF_INPUT_SIZE, SPOOF_INPUT_SIZE])
        //             .toFloat()
        //             .div(255.0) // Normalize to [0,1]
        //             .expandDims(0);

        //         // Run anti-spoofing model
        //         const predictions = await spoofModel.predict(tensor);
        //         const score = predictions.dataSync()[0]; // Assuming single output
        //         tensor.dispose(); // Clean up memory

        //         // Determine real vs spoof
        //         isRealFace = score < SPOOF_THRESHOLD;
                
        //         // Draw anti-spoofing result
        //         ctx.fillStyle = isRealFace ? "green" : "red";
        //         ctx.font = "20px Montserrat sans-serif";
        //         ctx.fillText(`Live: ${score.toFixed(2)}`, 10, 30);
        //     }
        // } catch (error) {
        //     console.error('Anti-spoofing error:', error);
        // }

        // // Draw Challenge on Canvas
        // if (challenge) {
        //     ctx.font = "24px Montserrat sans-serif";
        //     ctx.fillStyle = "cyan";
        //     ctx.fillText(`Please ${challenge}`, 10, 80);
        // }

        // // Check for challenge timeout
        // if (challenge && (Date.now() - challengeStartTime > CHALLENGE_TIMEOUT)) {
        //     console.warn("Challenge failed: timeout. Potential spoof!");
        //     ctx.font = "20px Montserrat sans-serif";
        //     ctx.fillStyle = "red";
        //     ctx.fillText("Liveness check failed: timeout!", 10, 110);
        //     challenge = null; // Reset challenge
        //     challengeSuccess = false; // Reset challenge success
        // }

        // // Generate a new challenge if not already active
        // if (!challenge) {
        //     challenge = generateChallenge();
        //     challengeStartTime = Date.now();
        //     challengeSuccess = false; // Reset challenge success flag
        //     nodDetected = false; // Reset nod detection flag
        // }

        // // Challenge Success Detection
        // const landmarks = detection.landmarks;

        // // Challenge Success Detection - Head Nod
        // if (challenge === "nod") {
        //     const currentNose = detection.landmarks.getNose()[0]; // Get the tip of the nose
        //     headPositionHistory.push({ x: currentNose.x, y: currentNose.y });

        //     if (headPositionHistory.length > HEAD_NOD_HISTORY_LENGTH) {
        //         headPositionHistory.shift(); // Remove the oldest entry
        //     }

        //     if (headPositionHistory.length === HEAD_NOD_HISTORY_LENGTH) {
        //         // Calculate vertical movement (nod)
        //         let yDiffSum = 0;
        //         for (let i = 1; i < HEAD_NOD_HISTORY_LENGTH; i++) {
        //             yDiffSum += (headPositionHistory[i].y - headPositionHistory[i - 1].y); // Only Y-axis
        //         }
        //         const totalVerticalMovement = yDiffSum;

        //         if (Math.abs(totalVerticalMovement) > NOD_THRESHOLD) {
        //             console.log("Head nod detected!");
        //             nodDetected = true; // Nod detected
        //             challengeSuccess = true; // Set challenge success
        //         }
        //     }

        //     if (nodDetected) {
        //         console.log("Nod Liveness challenge passed!");
        //         challengeSuccess = true; // Set challenge success flag
        //         // challenge = null; // Reset challenge
        //         // nodDetected = false; // Reset nod
        //         // headPositionHistory = []; // Reset history
        //     }
        // }

        if (detection.detection.score > confidenceThreshold && isRealFace) { // Check confidence
            if (!loginSuccessful && challengeSuccess) { // ADDED challengeSuccess check
                Object.keys(referenceDescriptors).forEach(key => {
                    // for (const refDescriptor of referenceDescriptors) {
                        // console.log("refDescriptor: ", referenceDescriptors[key]);
                        // console.log("detection.descriptor: ", detection.descriptor);
                        const distance = calculateDistance(referenceDescriptors[key], detection.descriptor);
                        if (distance < distanceThreshold) {
                            // **CRITICAL: CLEAR INTERVAL FIRST**
                            clearInterval(intervalId);
                            console.log("Face recognition stopped after successful login.");

                            loginSuccessful = true; // ADDED: Set loginSuccessful

                            drawBox.options.label = 'Matched face';
                            drawBox.draw(canvas);

                            const obj = {
                                [key]: referenceDescriptors[key]
                              };

                            // Dispatch login event with the matched reference descriptor
                            const loginEvent = new CustomEvent('click', {
                                detail: {
                                    descriptor: JSON.stringify(obj), // Send only the matched descriptor
                                }
                            });

                            loginButton.dispatchEvent(loginEvent);

                            console.log("Login event dispatched!", loginEvent.detail);

                            // break; // Exit the for...of loop
                        }
                    // }
                })
            }

            // New face detected
            drawBox.options.label = 'New Face - Processing...';
            drawBox.draw(canvas);

            if (!isCollecting && !loginSuccessful && challengeSuccess) { // ADDED challengeSuccess check
                isCollecting = true;
                collectedDescriptors = []; // Reset the array
                console.log("Collecting new descriptors...");
            }

            if (isCollecting && collectedDescriptors.length < numDescriptorsToCollect && challengeSuccess) { // ADDED challengeSuccess check
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
            // Show spoof warning
            // if (!isRealFace) {
            //     ctx.fillStyle = "red";
            //     ctx.font = "20px Montserrat sans-serif";
            //     ctx.fillText("Potential spoof detected!", 10, 110);
            // }
            // Prevent login/registration on spoof
            return;
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



