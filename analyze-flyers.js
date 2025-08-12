import OpenAI from 'openai';
import fs from 'fs';
import path from 'path';

const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
});

function encodeImageToBase64(imagePath) {
  const imageBuffer = fs.readFileSync(imagePath);
  return imageBuffer.toString('base64');
}

async function analyzeFlyer(imagePath) {
  console.log(`\n🔍 Analyzing: ${path.basename(imagePath)}`);
  
  try {
    const base64Image = encodeImageToBase64(imagePath);
    
    const response = await openai.chat.completions.create({
      model: "gpt-4o",
      messages: [
        {
          role: "user",
          content: [
            {
              type: "text",
              text: `Please analyze this event flyer and extract the following details in JSON format:

{
  "eventName": "name of the event",
  "eventType": "type/category of event (concert, festival, party, etc.)",
  "date": "date of the event",
  "time": "time of the event",
  "location": "venue/location details",
  "address": "full address if available",
  "performers": ["list of performers/artists"],
  "ticketInfo": "ticket prices, sales info, etc.",
  "contactInfo": "phone, website, social media",
  "additionalDetails": "any other relevant information"
}

If any information is not clearly visible or available, use null for that field. Be as accurate as possible with the text you can read from the flyer.`
            },
            {
              type: "image_url",
              image_url: {
                url: `data:image/jpeg;base64,${base64Image}`
              }
            }
          ]
        }
      ],
      max_tokens: 1000
    });

    const content = response.choices[0].message.content;
    
    // Extract JSON from the response
    const jsonMatch = content.match(/\{[\s\S]*\}/);
    if (jsonMatch) {
      const eventData = JSON.parse(jsonMatch[0]);
      return eventData;
    } else {
      console.log("Raw response:", content);
      return { error: "Could not parse JSON from response" };
    }
    
  } catch (error) {
    console.error(`Error analyzing ${imagePath}:`, error.message);
    return { error: error.message };
  }
}

async function analyzeAllFlyers() {
  const testDataDir = './test-data';
  
  if (!process.env.OPENAI_API_KEY) {
    console.error('❌ Please set your OPENAI_API_KEY environment variable');
    console.log('💡 Run: export OPENAI_API_KEY="your-api-key-here"');
    process.exit(1);
  }

  try {
    const files = fs.readdirSync(testDataDir);
    const imageFiles = files.filter(file => 
      /\.(jpg|jpeg|png|gif|bmp|webp)$/i.test(file)
    );

    if (imageFiles.length === 0) {
      console.log('❌ No image files found in test-data directory');
      return;
    }

    console.log(`📁 Found ${imageFiles.length} image(s) to analyze`);
    
    const results = [];
    
    for (const file of imageFiles) {
      const filePath = path.join(testDataDir, file);
      const result = await analyzeFlyer(filePath);
      
      results.push({
        filename: file,
        ...result
      });
      
      console.log('✅ Results:');
      console.log(JSON.stringify(result, null, 2));
    }

    // Save all results to a file
    const outputFile = 'analysis-results.json';
    fs.writeFileSync(outputFile, JSON.stringify(results, null, 2));
    console.log(`\n💾 All results saved to ${outputFile}`);
    
  } catch (error) {
    console.error('Error reading test-data directory:', error.message);
  }
}

// Run the analysis
analyzeAllFlyers();