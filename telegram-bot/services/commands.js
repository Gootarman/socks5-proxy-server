import passwordGenerator from 'generate-password'
import moment from 'moment'
import 'moment/locale/ru.js'
import _ from 'lodash'

import { USER_STATE } from './constants.js'
import { getChatIdAndUserName } from './utils.js'

function getProxyHostFromPublicUrl () {
  const publicUrl = process.env.PUBLIC_URL?.trim()

  if (!publicUrl) {
    return ''
  }

  try {
    const normalizedUrl = /^https?:\/\//.test(publicUrl) ? publicUrl : `http://${publicUrl}`

    return new URL(normalizedUrl).hostname
  } catch {
    return publicUrl
      .replace(/^https?:\/\//, '')
      .split('/')[0]
      .split(':')[0]
  }
}

function escapeHtml (str = '') {
  return str
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function getMiniAppUrl () {
  const miniAppUrl = process.env.MINI_APP_URL?.trim()
  if (miniAppUrl) {
    return miniAppUrl
  }

  const publicUrl = process.env.PUBLIC_URL?.trim()
  if (!publicUrl) {
    return null
  }

  return `${publicUrl.replace(/\/$/, '')}/mini-app`
}

function isMiniAppUrlAllowed (miniAppUrl) {
  if (!miniAppUrl) {
    return false
  }

  try {
    const parsedUrl = new URL(miniAppUrl)
    if (parsedUrl.protocol === 'https:') {
      return true
    }

    return parsedUrl.protocol === 'http:' && ['localhost', '127.0.0.1'].includes(parsedUrl.hostname)
  } catch {
    return false
  }
}

function getAdminReplyMarkup () {
  const keyboard = []
  const miniAppUrl = getMiniAppUrl()

  if (isMiniAppUrlAllowed(miniAppUrl)) {
    keyboard.push([{
      text: 'Открыть Mini App',
      web_app: { url: miniAppUrl }
    }])
  }

  keyboard.push(
    ['/users_stats', '/get_users'],
    ['/create_user', '/delete_user'],
    ['/generate_pass']
  )

  return {
    keyboard,
    resize_keyboard: true
  }
}

function getActionFromWebAppData (rawData) {
  if (!rawData) {
    return null
  }

  try {
    const parsed = JSON.parse(rawData)
    if (!parsed?.action) {
      return null
    }

    return {
      action: parsed.action,
      length: parsed.length
    }
  } catch {
    return null
  }
}

moment.locale('ru')

export default function ({ bot, logger, store }) {
  const miniAppUrl = getMiniAppUrl()
  const isMiniAppEnabled = isMiniAppUrlAllowed(miniAppUrl)
  const adminReplyMarkup = getAdminReplyMarkup()
  const removeKeyboard = { remove_keyboard: true }

  const isAdminOrReply = async (chatId, username) => {
    if (!await store.isAdmin(username)) {
      await bot.sendMessage(chatId, 'Извините, эта функция доступна только администраторам.')
      return false
    }

    return true
  }

  const handleUsersStats = async (chatId, username) => {
    const dataUsage = await store.getUsersStats()

    let message = '<b>Статистика трафика пользователей:</b>\n\n'

    if (dataUsage.length > 0) {
      dataUsage.forEach(u => {
        message += `<b>${u[0]}.</b> ${u[1]} (${moment(u[4]).fromNow()}): ${u[3]}\n`
      })
    } else {
      message += 'Статистика пока отсутствует.'
    }

    await store.setUserState(username, { state: USER_STATE.IDLE, data: {} })
    await bot.sendMessage(chatId, message, {
      parse_mode: 'HTML',
      reply_markup: adminReplyMarkup
    })
  }

  const handleCreateUser = async (chatId, username) => {
    const userState = { state: USER_STATE.CREATE_USER_ENTER_USERNAME, data: {} }

    await store.setUserState(username, userState)
    await bot.sendMessage(chatId, 'Введите логин для нового пользователя прокси.', {
      reply_markup: removeKeyboard
    })
  }

  const handleDeleteUser = async (chatId, username) => {
    const userState = { state: USER_STATE.DELETE_USER_ENTER_USERNAME, data: {} }
    await store.setUserState(username, userState)
    await bot.sendMessage(chatId, 'Введите логин пользователя для удаления.', {
      reply_markup: removeKeyboard
    })
  }

  const handleGetUsers = async (chatId, username) => {
    await store.setUserState(username, { state: USER_STATE.IDLE, data: {} })
    const users = await store.getUsers()

    let message = 'Пользователей нет.'

    if (users.length > 0) {
      message = '<b>Пользователи</b>:\n\n'

      users.sort().forEach((u, i) => {
        message += `${i + 1}. ${u}\n`
      })

      message += `\n<b>Итого: ${users.length}</b>`
    }

    await bot.sendMessage(chatId, message, {
      parse_mode: 'HTML',
      reply_markup: adminReplyMarkup
    })
  }

  const handleGeneratePassword = async (chatId, rawLength) => {
    const length = parseInt(String(rawLength).trim()) || 10

    await bot.sendMessage(chatId, passwordGenerator.generate({
      length,
      numbers: true,
      uppercase: true,
      strict: true
    }), {
      reply_markup: adminReplyMarkup
    })
  }

  const handleWebAppAction = async (chatId, username, rawWebAppData) => {
    const data = getActionFromWebAppData(rawWebAppData)

    if (!data) {
      await bot.sendMessage(chatId, 'Не удалось обработать данные из Mini App.', {
        reply_markup: adminReplyMarkup
      })

      return
    }

    switch (data.action) {
      case 'users_stats':
        await handleUsersStats(chatId, username)
        break
      case 'create_user':
        await handleCreateUser(chatId, username)
        break
      case 'delete_user':
        await handleDeleteUser(chatId, username)
        break
      case 'get_users':
        await handleGetUsers(chatId, username)
        break
      case 'generate_pass':
        await handleGeneratePassword(chatId, data.length)
        break
      default:
        await bot.sendMessage(chatId, 'Неизвестное действие Mini App.', {
          reply_markup: adminReplyMarkup
        })
    }
  }

  bot.onText(/\/start(.*)/, async (msg, _match) => {
    const { chatId, username } = getChatIdAndUserName(msg)

    logger.debug(`Received start message from @${username}`)

    try {
      if (!await store.isAdmin(username)) {
        await bot.sendMessage(chatId, 'Здравствуйте! Вы не являетесь администратором этого прокси-сервера.')

        return
      }

      const userState = { state: USER_STATE.IDLE, data: {} }
      await store.setUserState(username, userState)

      await Promise.all([
        bot.sendMessage(chatId, isMiniAppEnabled
          ? 'Здравствуйте! Вы можете управлять прокси-сервером через команды или Mini App.'
          : 'Здравствуйте! Вы можете управлять прокси-сервером через команды.',
        {
          reply_markup: adminReplyMarkup
        }),
        store.updateAdminChatId(username, chatId)
      ])
    } catch (err) {
      logger.error(err)
    }
  })

  bot.onText(/\/users_stats(.*)/, async (msg, _match) => {
    const { chatId, username } = getChatIdAndUserName(msg)

    logger.debug(`Received stats request from @${username}`)
    try {
      if (!await isAdminOrReply(chatId, username)) {
        return
      }

      await handleUsersStats(chatId, username)
    } catch (err) {
      logger.error(err)
      await bot.sendMessage(chatId, err.message, { reply_markup: adminReplyMarkup })
    }
  })

  bot.onText(/\/create_user(.*)/, async (msg, match) => {
    const { chatId, username } = getChatIdAndUserName(msg)

    logger.debug(`Received create user request from @${username}`)

    try {
      logger.debug(`Match: ${JSON.stringify(match)}`)
      if (!await isAdminOrReply(chatId, username)) {
        return
      }

      await handleCreateUser(chatId, username)
    } catch (err) {
      logger.error(err)
      await bot.sendMessage(chatId, err.message, { reply_markup: adminReplyMarkup })
    }
  })

  bot.onText(/\/delete_user(.*)/, async (msg, _match) => {
    const { chatId, username } = getChatIdAndUserName(msg)

    logger.debug(`Received create user request from @${username}`)

    try {
      if (!await isAdminOrReply(chatId, username)) {
        return
      }

      await handleDeleteUser(chatId, username)
    } catch (err) {
      logger.error(err)
      await bot.sendMessage(chatId, err.message, { reply_markup: adminReplyMarkup })
    }
  })

  bot.onText(/\/get_users(.*)/, async (msg, _match) => {
    const { chatId, username } = getChatIdAndUserName(msg)

    logger.debug(`Received get users request from @${username}`)

    try {
      if (!await isAdminOrReply(chatId, username)) {
        return
      }

      await handleGetUsers(chatId, username)
    } catch (err) {
      logger.error(err)
      await bot.sendMessage(chatId, err.message, { reply_markup: adminReplyMarkup })
    }
  })

  bot.onText(/\/generate_pass(.*)/, async (msg, match) => {
    const { chatId, username } = getChatIdAndUserName(msg)

    try {
      if (!await isAdminOrReply(chatId, username)) {
        return
      }

      await handleGeneratePassword(chatId, match[1])
    } catch (err) {
      logger.error(err)
      await bot.sendMessage(chatId, err.message, { reply_markup: adminReplyMarkup })
    }
  })

  bot.on('message', async msg => {
    const webAppData = msg.web_app_data?.data
    if (!webAppData) {
      return
    }

    const { chatId, username } = getChatIdAndUserName(msg)
    logger.debug(`Received web app action from @${username}: ${webAppData}`)

    try {
      if (!await isAdminOrReply(chatId, username)) {
        return
      }

      await handleWebAppAction(chatId, username, webAppData)
    } catch (err) {
      logger.error(err)
      await bot.sendMessage(chatId, err.message, { reply_markup: adminReplyMarkup })
    }
  })

  // eslint-disable-next-line
    bot.onText(/^[^\/].*/, async (msg, _match) => {
    const { chatId, username } = getChatIdAndUserName(msg)

    try {
      const userState = await store.getUserState(username)

      if (_.isNull(userState)) {
        logger.debug('User state is idle')

        return
      }

      switch (userState.state) {
        case USER_STATE.IDLE:
          await bot.sendMessage(chatId, 'Введите команду.', {
            reply_markup: adminReplyMarkup
          })
          break
        case USER_STATE.CREATE_USER_ENTER_USERNAME: {
          const proxyUsername = msg.text.trim()

          logger.debug(`Entered username: ${proxyUsername}`)

          if (!proxyUsername) {
            await bot.sendMessage(chatId, 'Логин не может быть пустым. Введите другой.')

            break
          }

          if (!await store.isUsernameFree(proxyUsername)) {
            await bot.sendMessage(chatId, 'Этот логин уже занят. Введите другой.')

            break
          }

          userState.state = USER_STATE.CREATE_USER_ENTER_PASSWORD
          userState.data.username = proxyUsername

          const suggestedPassword = passwordGenerator.generate({
            length: 10,
            numbers: true,
            uppercase: true,
            strict: true
          })

          await store.setUserState(username, userState)
          await bot.sendMessage(chatId, 'Введите пароль или используйте предложенный вариант.', {
            reply_markup: {
              keyboard: [[suggestedPassword]]
            }
          })

          break
        }
        case USER_STATE.CREATE_USER_ENTER_PASSWORD: {
          const proxyPassword = msg.text.trim()

          if (!proxyPassword) {
            await bot.sendMessage(chatId, 'Пароль не может быть пустым. Введите другой.')

            break
          }

          await store.createUser(userState.data.username, proxyPassword)
          await store.setUserState(username, { state: USER_STATE.IDLE, data: {} })

          const proxyHost = getProxyHostFromPublicUrl()
          const proxyPort = process.env.APP_PORT
          const encodedProxyHost = encodeURIComponent(proxyHost)
          const encodedProxyPort = encodeURIComponent(proxyPort)
          const encodedProxyUsername = encodeURIComponent(userState.data.username)
          const encodedProxyPassword = encodeURIComponent(proxyPassword)

          const telegramProxyLink = proxyHost
            ? `tg://socks?server=${encodedProxyHost}&port=${encodedProxyPort}&user=${encodedProxyUsername}&pass=${encodedProxyPassword}`
            : null

          const messageParts = []

          if (telegramProxyLink) {
            messageParts.push('<b>Для подключения прокси нажмите ссылку</b> - это быстрый вариант.')
            messageParts.push('')

            messageParts.push(`<a href="${escapeHtml(telegramProxyLink)}">Подключить в Telegram</a>`)

            messageParts.push('')
          }

          messageParts.push('<b>Ручная настройка</b>')

          if (proxyHost) {
            messageParts.push(`<b>хост:</b> <code>${escapeHtml(proxyHost)}</code>`)
          } else {
            messageParts.push('<i>Хост прокси не настроен.</i>')
          }

          messageParts.push(`<b>порт:</b> <code>${escapeHtml(proxyPort)}</code>`)
          messageParts.push(`<b>логин:</b> <code>${escapeHtml(userState.data.username)}</code>`)
          messageParts.push(`<b>пароль:</b> <code>${escapeHtml(proxyPassword)}</code>`)

          const message = messageParts.join('\n')

          await bot.sendMessage(chatId, message, {
            parse_mode: 'HTML',
            reply_markup: adminReplyMarkup
          })

          break
        }
        case USER_STATE.DELETE_USER_ENTER_USERNAME: {
          const usernameToDelete = msg.text.trim()

          logger.debug(`Entered username: ${usernameToDelete}`)

          if (await store.isUsernameFree(usernameToDelete)) {
            await bot.sendMessage(chatId, 'Пользователь с таким логином не существует. Введите другой.')

            break
          }

          await store.deleteUser(usernameToDelete)
          await store.setUserState(username, { state: USER_STATE.IDLE, data: {} })
          await bot.sendMessage(chatId, 'Пользователь удалён.', {
            reply_markup: adminReplyMarkup
          })

          break
        }
      }
    } catch (err) {
      logger.error(err)
      await bot.sendMessage(chatId, err.message, { reply_markup: adminReplyMarkup })
    }
  })
}
