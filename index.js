const crypto = require('crypto');
const express = require('express');
const axios = require('axios');

const config = require('./node-config.json');

function sign() {
    const t = Date.now();
    const nonce = "1";
    const data = config.switchbotToken + t + nonce;
    const signTerm = crypto.createHmac('sha256', config.switchbotSecret)
        .update(Buffer.from(data, 'utf-8'))
        .digest();
    const sign = signTerm.toString("base64");
    return {
        sign,
        nonce,
        t
    };
}

const app = express();

const actionMap = {
    entry: 'turnOn',
    garage: 'turnOff'
}

app.route(`/bot/:action`)
    .all((req, res, next) => {
        res.set({
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Allow-Credentials": "true",
            "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept, Authorization, x-access-token",
            "Access-Control-Expose-Headers": "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type",
            "Access-Control-Allow-Methods": "POST, OPTIONS, GET, PUT"
        });
        if(req.method == 'OPTIONS')
            res.sendStatus(204);
        else
            next();
    })
    .post(async (req, res) => {
        const cmd = actionMap[req.params.action];
        console.log('Bot action ' + cmd);
        if(!cmd) {
            res.sendStatus(400);
            return;
        }
        try {
            await bot(cmd);
            await new Promise(resolve => setTimeout(resolve, 2000));
            res.status(200).send({
                ok: true
            });
        } catch(err) {
            res.status(500).send({
                status: err.response.status,
                data: err.response.data
            });
        }
    });

function bot(cmd) {
    return axios.request({
        url: `https://api.switch-bot.com/v1.1/devices/${config.switchbotDeviceId}/commands`,
        method: 'POST',
        headers: {
            Authorization: config.switchbotToken,
            ...sign()
        },
        data: {
            command: cmd,
            parameter: "default",
            commandType: "command"
        }
    });
}

app.listen(config.httpPort, () => console.log('Server started on port ' + config.httpPort));