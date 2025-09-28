# node 20 is lts at the time of writing
FROM node:lts-alpine

# Create app directory
WORKDIR /usr/src/app

# Download and install kepubify
RUN wget https://github.com/pgaskin/kepubify/releases/download/v4.0.4/kepubify-linux-64bit && \
    mv kepubify-linux-64bit /usr/local/bin/kepubify && \
    chmod +x /usr/local/bin/kepubify



# Copy files needed by npm install
COPY package*.json ./

# Install app dependencies
RUN npm install --omit=dev

# Copy the rest of the app files (see .dockerignore)
COPY . ./

# Create uploads directory if it doesn't exist
RUN mkdir uploads

EXPOSE 3001
CMD [ "npm", "start" ]
